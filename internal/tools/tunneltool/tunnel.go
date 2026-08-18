package tunneltool

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// connectTimeout SSH 建连超时
const connectTimeout = 10 * time.Second

// buildAuth 按认证方式构造 ssh.AuthMethod 列表
func buildAuth(cfg TunnelConfig) ([]ssh.AuthMethod, error) {
	switch cfg.AuthMethod {
	case AuthPassword:
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil

	case AuthAgent:
		// 代理 = 不指定身份时的默认 ssh 行为：先问本机 ssh-agent，
		// agent 没有可用密钥再读默认密钥文件
		var methods []ssh.AuthMethod
		if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
			if conn, err := net.Dial("unix", sock); err == nil {
				signers, err := agent.NewClient(conn).Signers()
				if err == nil && len(signers) > 0 {
					methods = append(methods, ssh.PublicKeys(signers...))
				} else {
					_ = conn.Close()
				}
			}
		}
		if def, err := defaultKeySigners(); err == nil && len(def) > 0 {
			methods = append(methods, ssh.PublicKeys(def...))
		}
		if len(methods) == 0 {
			return nil, fmt.Errorf("未找到可用身份：ssh-agent 无密钥且 %s 下无默认密钥文件", sshDir())
		}
		return methods, nil

	case AuthPublicKey:
		path := expandTilde(cfg.KeyPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取密钥失败: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("解析密钥失败: %w（支持 OpenSSH/PEM 格式私钥）", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil

	default:
		return nil, fmt.Errorf("未知认证方式: %s", cfg.AuthMethod)
	}
}

func sshDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.ssh"
	}
	return filepath.Join(home, ".ssh")
}

// defaultKeySigners 读取默认密钥文件（等价 ssh 不指定 IdentityFile 时的行为）
func defaultKeySigners() ([]ssh.Signer, error) {
	names := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	var signers []ssh.Signer
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(sshDir(), n))
		if err != nil {
			continue
		}
		if s, err := ssh.ParsePrivateKey(data); err == nil {
			signers = append(signers, s)
		}
	}
	return signers, nil
}

// expandTilde 展开 ~ 前缀到家目录
func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// dial 建立 SSH 连接（认证/网络错误统一分类为可读中文）
func dial(cfg TunnelConfig) (*ssh.Client, error) {
	auth, err := buildAuth(cfg)
	if err != nil {
		return nil, err
	}
	clientConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // v1 简化：不做 known_hosts 校验
		Timeout:         connectTimeout,
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), clientConfig)
	if err != nil {
		return nil, classifyDialError(err)
	}
	return client, nil
}

// classifyDialError 把 ssh.Dial 错误归类为可读中文
func classifyDialError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "ssh: unable to authenticate"),
		strings.Contains(msg, "no supported methods remain"):
		return fmt.Errorf("认证失败：用户名/密码/密钥被服务器拒绝")
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("网络不通：连接被拒绝（主机或 SSH 端口不可达）")
	case strings.Contains(msg, "i/o timeout"), strings.Contains(msg, "connection timed out"):
		return fmt.Errorf("网络不通：连接超时")
	case strings.Contains(msg, "no such host"):
		return fmt.Errorf("网络不通：主机名无法解析")
	default:
		return fmt.Errorf("连接失败: %s", msg)
	}
}

// forward 在已建立的 SSH 连接上做本地端口转发。
// accept 循环里每个连接两个 goroutine 双向 io.Copy 桥接；
// 任何一方断开即关闭双方。返回的 error 非 nil 时表示转发异常（掉线）。
func forward(client *ssh.Client, cfg TunnelConfig, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err // listener 关闭（正常停止）或异常
		}
		go func(c net.Conn) {
			defer c.Close()
			remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", cfg.RemoteHost, cfg.RemotePort))
			if err != nil {
				return // 单个连接的远端失败不致命（可能是端口没起）
			}
			defer remote.Close()
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(remote, c); done <- struct{}{} }()
			go func() { _, _ = io.Copy(c, remote); done <- struct{}{} }()
			<-done // 任一方向结束即收尾
		}(conn)
	}
}

// testTunnel 三步验证：能连 + 能认证 + 远程端口可达（不保持转发）
func testTunnel(cfg TunnelConfig) error {
	// 本地端口占用预检（不实际监听，dial 探测）
	if ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.LocalHost, cfg.LocalPort)); err != nil {
		return fmt.Errorf("本地端口 %d 已被占用", cfg.LocalPort)
	} else {
		_ = ln.Close()
	}
	client, err := dial(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", cfg.RemoteHost, cfg.RemotePort))
	if err != nil {
		return fmt.Errorf("远程端口不通：%s:%d 无法连接", cfg.RemoteHost, cfg.RemotePort)
	}
	_ = remote.Close()
	return nil
}

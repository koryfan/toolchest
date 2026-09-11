package tunneltool

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// generateTestHostKey 生成临时主机密钥（mock server 用）
func generateTestHostKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// mockSSHServer 本地 mock SSH 服务端：密码认证 + direct-tcpip 转发到本地目标端口
type mockSSHServer struct {
	ln       net.Listener
	targetLn net.Listener // 转发目标（echo 服务）
	addr     string
}

func newMockSSHServer(t *testing.T, user, pass string) *mockSSHServer {
	t.Helper()

	// echo 目标服务
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := targetLn.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_, _ = io.Copy(c, c) // echo
			}()
		}
	}()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if c.User() == user && string(password) == pass {
				return nil, nil
			}
			return nil, io.EOF // 拒绝
		},
	}
	// 生成临时主机密钥
	key, _ := generateTestHostKey()
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	cfg.AddHostKey(signer)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleSSHConn(conn, cfg, targetLn)
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		_ = targetLn.Close()
	})
	return &mockSSHServer{ln: ln, targetLn: targetLn, addr: ln.Addr().String()}
}

// handleSSHConn 处理一条 SSH 连接：接受 direct-tcpip 通道并桥到目标
func handleSSHConn(conn net.Conn, cfg *ssh.ServerConfig, target net.Listener) {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		switch newChan.ChannelType() {
		case "direct-tcpip":
			handleDirectTCPIP(newChan, target)
		case "session":
			handleMockExec(newChan, target) // mock：exec 命令不真执行，stdio 桥到 echo 目标（等价 nc）
		default:
			_ = newChan.Reject(ssh.UnknownChannelType, "unsupported")
		}
	}
}

// handleDirectTCPIP 接受 direct-tcpip 通道并桥到目标
func handleDirectTCPIP(newChan ssh.NewChannel, target net.Listener) {
	var payload struct {
		Addr           string
		Port           uint32
		OriginatorAddr string
		OriginatorPort uint32
	}
	if err := ssh.Unmarshal(newChan.ExtraData(), &payload); err != nil {
		_ = newChan.Reject(ssh.Prohibited, "bad payload")
		return
	}
	ch, requests, err := newChan.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(requests)

	targetAddr := target.Addr().String() // mock：忽略 payload 指定地址，统一转发到 echo 目标
	dest, err := net.Dial("tcp", targetAddr)
	if err != nil {
		_ = ch.Close()
		return
	}
	go func() {
		done := make(chan struct{}, 2)
		go func() { _, _ = io.Copy(dest, ch); done <- struct{}{} }()
		go func() { _, _ = io.Copy(ch, dest); done <- struct{}{} }()
		<-done
		_ = ch.Close()
		_ = dest.Close()
	}()
}

// handleMockExec 接受 session 通道：exec 请求不真执行命令，
// 直接把 stdio 桥到 echo 目标（模拟远端 nc 行为，验证 exec 隧道转发）
func handleMockExec(newChan ssh.NewChannel, target net.Listener) {
	ch, requests, err := newChan.Accept()
	if err != nil {
		return
	}
	go func() {
		defer ch.Close()
		for req := range requests {
			if req.Type != "exec" {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				continue
			}
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			dest, err := net.Dial("tcp", target.Addr().String())
			if err != nil {
				return
			}
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(dest, ch); done <- struct{}{} }()
			go func() { _, _ = io.Copy(ch, dest); done <- struct{}{} }()
			<-done
			status := struct{ Status uint32 }{}
			_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(status))
			_ = dest.Close()
			return
		}
	}()
}

func mockAddrPort(t *testing.T, addr string) (host string, port int) {
	t.Helper()
	h, pStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	var p int
	if err := json.Unmarshal([]byte(pStr), &p); err != nil {
		t.Fatal(err)
	}
	return h, p
}

func freeLocalPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// TestStartAndForward 端到端：Start 后本地连接经隧道转发到 echo 目标
func TestStartAndForward(t *testing.T) {
	srv := newMockSSHServer(t, "root", "s3cret")
	host, sshPort := mockAddrPort(t, srv.addr)
	localPort := freeLocalPort(t)

	cfg := TunnelConfig{
		Name: "测试隧道", Host: host, Port: sshPort, Username: "root",
		AuthMethod: AuthPassword, Password: "s3cret",
		LocalHost: "127.0.0.1", LocalPort: localPort,
		RemoteHost: "localhost", RemotePort: 12345,
	}

	m := NewManager(nil)
	m.ReloadConfigs([]TunnelConfig{cfg})
	if err := m.Start(cfg.Name); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop(cfg.Name)

	// 等本地监听就绪
	deadline := time.Now().Add(2 * time.Second)
	var conn net.Conn
	var err error
	for time.Now().Before(deadline) {
		conn, err = net.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", localPort)))
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("本地转发端口连不上: %v", err)
	}
	defer conn.Close()

	msg := "hello-through-tunnel\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(msg))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("读回显失败: %v", err)
	}
	if string(buf) != msg {
		t.Fatalf("回显不符: %q", string(buf))
	}
}

// TestStartForwardExec exec 模式端到端：Start 后本地连接经 session stdio 中继到 echo 目标
func TestStartForwardExec(t *testing.T) {
	srv := newMockSSHServer(t, "root", "s3cret")
	host, sshPort := mockAddrPort(t, srv.addr)
	localPort := freeLocalPort(t)

	cfg := TunnelConfig{
		Name: "exec隧道", Host: host, Port: sshPort, Username: "root",
		AuthMethod: AuthPassword, Password: "s3cret",
		LocalHost: "127.0.0.1", LocalPort: localPort,
		RemoteHost: "localhost", RemotePort: 12345,
		ForwardMode: ForwardExec,
	}

	m := NewManager(nil)
	m.ReloadConfigs([]TunnelConfig{cfg})
	if err := m.Start(cfg.Name); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop(cfg.Name)

	// 等本地监听就绪
	deadline := time.Now().Add(2 * time.Second)
	var conn net.Conn
	var err error
	for time.Now().Before(deadline) {
		conn, err = net.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", localPort)))
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("本地转发端口连不上: %v", err)
	}
	defer conn.Close()

	msg := "hello-through-exec-tunnel\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(msg))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("读回显失败: %v", err)
	}
	if string(buf) != msg {
		t.Fatalf("回显不符: %q", string(buf))
	}
}

// TestStartAuthFail 认证失败分类
func TestStartAuthFail(t *testing.T) {
	srv := newMockSSHServer(t, "root", "right")
	host, sshPort := mockAddrPort(t, srv.addr)

	cfg := TunnelConfig{
		Name: "坏密码", Host: host, Port: sshPort, Username: "root",
		AuthMethod: AuthPassword, Password: "wrong",
		LocalHost: "127.0.0.1", LocalPort: freeLocalPort(t),
		RemoteHost: "localhost", RemotePort: 12345,
	}
	m := NewManager(nil)
	m.ReloadConfigs([]TunnelConfig{cfg})
	err := m.Start(cfg.Name)
	if err == nil {
		t.Fatal("错误密码不应连上")
	}
	if !strings.Contains(err.Error(), "认证失败") {
		t.Fatalf("错误应分类为认证失败: %v", err)
	}
}

// TestClassifyDialError 错误分类表驱动
func TestClassifyDialError(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"认证", "ssh: unable to authenticate", "认证失败"},
		{"拒绝", "dial tcp 1.2.3.4:22: connect: connection refused", "网络不通"},
		{"超时", "dial tcp 1.2.3.4:22: i/o timeout", "网络不通"},
		{"域名", "dial tcp: lookup nope.invalid: no such host", "网络不通"},
		{"其他", "some other error", "连接失败"},
	}
	for _, c := range cases {
		got := classifyDialError(errString(c.in)).Error()
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want 包含 %q", c.name, got, c.want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// TestConfigValidateNormalize 校验与规范化
func TestConfigValidateNormalize(t *testing.T) {
	if msg := (TunnelConfig{}).validate(); msg != "隧道名称不能为空" {
		t.Errorf("空配置校验: %q", msg)
	}
	full := TunnelConfig{Name: "n", Host: "h", Username: "u", LocalPort: 1, RemotePort: 2, AuthMethod: AuthPassword, Password: "p"}
	if msg := full.validate(); msg != "" {
		t.Errorf("完整配置不应报错: %q", msg)
	}
	got := TunnelConfig{Name: "n", Host: "h", Username: "u", LocalPort: 1, RemotePort: 2, AuthMethod: AuthPassword, Password: "p"}.normalize()
	if got.Port != 22 || got.LocalHost != "127.0.0.1" || got.RemoteHost != "localhost" {
		t.Errorf("默认值规范化不符: %#v", got)
	}
	if got := (TunnelConfig{Name: "n"}).normalize().ForwardMode; got != ForwardDirect {
		t.Errorf("forwardMode 默认值: %q", got)
	}
	if got := (TunnelConfig{RemoteHost: "10.0.0.1", RemotePort: 5432}).execRemoteCommand(); got != "nc 10.0.0.1 5432" {
		t.Errorf("默认 exec 命令: %q", got)
	}
	if got := (TunnelConfig{RemoteHost: "h", RemotePort: 1, ExecCommand: "socat - TCP:h:1"}).execRemoteCommand(); got != "socat - TCP:h:1" {
		t.Errorf("自定义 exec 命令优先: %q", got)
	}
}

// TestToolSaveListDelete Tool 层 op 流转
func TestToolSaveListDelete(t *testing.T) {
	tl := New(DefaultState(), nil)

	// save
	form, _ := json.Marshal(TunnelConfig{
		Name: "t1", Host: "h", Username: "u", AuthMethod: AuthAgent,
		LocalPort: 10001, RemotePort: 80,
	})
	r := tl.Invoke("save", string(form))
	if r.IsError {
		t.Fatalf("save: %s", r.Notice)
	}
	// list
	r = tl.Invoke("list", "")
	data, ok := r.Data.(payload)
	if !ok || len(data.Tunnels) != 1 || data.Tunnels[0].Name != "t1" {
		t.Fatalf("list 不符: %#v", r.Data)
	}
	if data.Tunnels[0].Status != StatusStopped || data.Tunnels[0].Port != 22 || data.Tunnels[0].LocalHost != "127.0.0.1" {
		t.Fatalf("save 后未规范化: %#v", data.Tunnels[0])
	}
	// 编辑（同名覆盖）
	form2, _ := json.Marshal(TunnelConfig{
		Name: "t1", Host: "h2", Username: "u", AuthMethod: AuthAgent,
		LocalPort: 10001, RemotePort: 81,
	})
	if r := tl.Invoke("save", string(form2)); r.IsError {
		t.Fatalf("save2: %s", r.Notice)
	}
	if st := tl.State().(State); len(st.Tunnels) != 1 || st.Tunnels[0].Host != "h2" {
		t.Fatalf("同名应覆盖: %#v", st.Tunnels)
	}
	// delete
	if r := tl.Invoke("delete", "t1"); r.IsError {
		t.Fatalf("delete: %s", r.Notice)
	}
	if st := tl.State().(State); len(st.Tunnels) != 0 {
		t.Fatalf("删除后应清空: %#v", st.Tunnels)
	}
	if r := tl.Invoke("delete", "不存在"); !r.IsError {
		t.Fatal("删除不存在的隧道应报错")
	}
}

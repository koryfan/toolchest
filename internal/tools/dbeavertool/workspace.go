// Package dbeavertool DBeaver 凭据查看工具。
//
// DDD Lite 边界：查询型工具，无跨实体不变量，不立聚合根——
// 领域层 = 纯函数组（Workspace 打开语义：解密→解析，失败即整体失败）；
// CredentialEntry 为值对象；目录记忆是应用层便利状态，留在 Tool（State）。
package dbeavertool

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// 领域错误：调用方（应用层/前端）据此区分提示文案
var (
	// ErrNotCredentials 文件存在但不是有效凭据密文（大小/padding 校验不通过）
	ErrNotCredentials = errors.New("不是有效的 DBeaver 凭据文件（可能损坏或加密参数不一致）")
	// ErrNotWorkspace 目录下没有 .dbeaver/credentials-config.json
	ErrNotWorkspace = errors.New("目录下没有 .dbeaver/credentials-config.json")
	// ErrBadContent 解密成功但内容不是合法凭据 JSON
	ErrBadContent = errors.New("解密后的内容不是合法的凭据 JSON")
)

// 加密参数（DBeaver 社区公开的固定值）：AES-128-CBC，key 固定，iv 全零，
// 明文前有 16 字节随机前缀。
var (
	cryptKey, _ = hex.DecodeString("babb4a9f774ab853c96c2d653dfe544a")
	cryptIV     = make([]byte, aes.BlockSize)
)

// CredentialEntry 值对象：单条连接凭据（不可变查询结果）。
// Name/Driver/Host 来自同目录 data-sources.json（明文）按 ID 关联，缺失时回退 ID。
type CredentialEntry struct {
	ID     string            `json:"id"`     // 连接标识，如 mysql8-193ab26b9e4-2527dc101f9196b
	Name   string            `json:"name"`   // 用户可见名称，如 "logs"
	Driver string            `json:"driver"` // 驱动，如 mysql8 / com_clickhouse
	Host   string            `json:"host"`   // 主机（来自 data-sources.json configuration）
	Keys   map[string]string `json:"keys"`   // 凭据字段（user/password/...），非字符串字段不展示
}

// Decrypt 解密 credentials-config.json 密文。
// 等价于：openssl aes-128-cbc -d -K <key> -iv 000..0（含 PKCS7 去除 + 跳过 16 字节随机前缀）。
func Decrypt(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w：文件大小 %d 不是 16 字节的整数倍", ErrNotCredentials, len(data))
	}
	block, err := aes.NewCipher(cryptKey)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, cryptIV).CryptBlocks(out, data)

	// PKCS7 padding 校验（与 openssl bad decrypt 同判据）
	pad := int(out[len(out)-1])
	if pad == 0 || pad > aes.BlockSize || !bytes.Equal(out[len(out)-pad:], bytes.Repeat([]byte{byte(pad)}, pad)) {
		return nil, ErrNotCredentials
	}
	out = out[:len(out)-pad]

	if len(out) < 16 {
		return nil, fmt.Errorf("%w：解密后内容过短", ErrNotCredentials)
	}
	return out[16:], nil // 跳过随机前缀
}

// Parse 解析解密后的明文 JSON。实际结构（无外层包装）：
// {"<连接ID>": {"#connection": {"user": "...", "password": "..."}}}
func Parse(plain []byte) ([]CredentialEntry, error) {
	var cfg map[string]map[string]json.RawMessage
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadContent, err)
	}
	entries := make([]CredentialEntry, 0, len(cfg))
	for id, sections := range cfg {
		keys := map[string]string{}
		for _, raw := range sections {
			var fields map[string]any
			if err := json.Unmarshal(raw, &fields); err != nil {
				continue // 非 {k:v} 结构的 section 跳过
			}
			for k, v := range fields {
				if s, ok := v.(string); ok {
					keys[k] = s
				}
			}
		}
		if len(keys) == 0 {
			continue // 空连接跳过
		}
		entries = append(entries, CredentialEntry{ID: id, Keys: keys})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%w：未解析出任何连接", ErrBadContent)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries, nil
}

// dataSources data-sources.json 的连接元信息（明文，非敏感）
type dataSourceInfo struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
}

// readDataSourceNames 读 data-sources.json 的 id → 名称/驱动/主机 映射。
// 文件缺失/损坏不致命：凭据仍可展示（回退 ID）。
func readDataSourceNames(dir string) map[string]dataSourceInfo {
	data, err := os.ReadFile(filepath.Join(dir, ".dbeaver", "data-sources.json"))
	if err != nil {
		return nil
	}
	var cfg struct {
		Connections map[string]struct {
			Name          string `json:"name"`
			Driver        string `json:"driver"`
			Configuration struct {
				Host string `json:"host"`
				Port any    `json:"port"` // 实际可能是字符串 "5432" 或数字
			} `json:"configuration"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	out := make(map[string]dataSourceInfo, len(cfg.Connections))
	for id, c := range cfg.Connections {
		info := dataSourceInfo{Name: c.Name, Driver: c.Driver, Host: c.Configuration.Host}
		switch p := c.Configuration.Port.(type) {
		case string:
			fmt.Sscanf(p, "%d", &info.Port)
		case float64:
			info.Port = int(p)
		}
		out[id] = info
	}
	return out
}

// OpenWorkspace 打开一个 DBeaver 工作目录：读凭据密文 → 解密 → 解析 →
// 关联 data-sources.json 元信息（名称/驱动/主机）。
// 不变量（由纯函数链天然保证）：返回的 Entries 必然是完整解密+解析成功的，
// 不存在半解密/部分解析状态——任一步失败整体失败。
func OpenWorkspace(dir string) ([]CredentialEntry, error) {
	data, err := os.ReadFile(filepath.Join(dir, ".dbeaver", "credentials-config.json"))
	if err != nil {
		return nil, fmt.Errorf("%w（需选到工作目录，如 General）", ErrNotWorkspace)
	}
	plain, err := Decrypt(data)
	if err != nil {
		return nil, err
	}
	entries, err := Parse(plain)
	if err != nil {
		return nil, err
	}

	// 关联名称（尽力而为，缺失回退 ID）
	names := readDataSourceNames(dir)
	for i := range entries {
		if info, ok := names[entries[i].ID]; ok {
			entries[i].Name = info.Name
			entries[i].Driver = info.Driver
			entries[i].Host = info.Host
		}
	}
	return entries, nil
}

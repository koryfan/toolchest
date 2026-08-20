package dbeavertool

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// encryptFixture 用产品同款参数加密一个明文 JSON，落盘为临时凭据文件
func encryptFixture(t *testing.T, dir string, plainJSON string) {
	t.Helper()
	block, _ := aes.NewCipher(cryptKey)

	// 明文 = 16 字节随机前缀 + JSON，PKCS7 补齐
	plain := append(make([]byte, 16), []byte(plainJSON)...)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < pad; i++ {
		plain = append(plain, byte(pad))
	}

	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, cryptIV).CryptBlocks(out, plain)

	dbDir := filepath.Join(dir, ".dbeaver")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dbDir, "credentials-config.json"), out, 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeDataSources 落一份 data-sources.json（port 为字符串，还原真实文件形态）
func writeDataSources(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".dbeaver", "data-sources.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenWorkspaceRoundtrip(t *testing.T) {
	dir := t.TempDir()
	encryptFixture(t, dir, `{
		"mysql8-abc-123": {"#connection": {"user": "phpapi", "password": "s3cret"}},
		"postgres-jdbc-xyz-9": {"#connection": {"user": "postgres", "password": "pg", "configuration": "flag"}},
		"empty-conn": {}
	}`)
	writeDataSources(t, dir, `{"connections": {
		"mysql8-abc-123": {"name": "订单库", "driver": "mysql8", "configuration": {"host": "db.example.com", "port": "3306"}},
		"postgres-jdbc-xyz-9": {"name": "日志库", "driver": "postgres-jdbc", "configuration": {"host": "pg.example.com"}}
	}}`)

	entries, err := OpenWorkspace(dir)
	if err != nil {
		t.Fatalf("decryptFile: %v", err)
	}
	if len(entries) != 2 { // empty-conn 无字段被跳过
		t.Fatalf("应解析出 2 条（空连接跳过）: %#v", entries)
	}
	if entries[0].ID != "mysql8-abc-123" || entries[0].Keys["user"] != "phpapi" || entries[0].Keys["password"] != "s3cret" {
		t.Fatalf("第一条不符: %#v", entries[0])
	}
	// data-sources.json 元信息关联
	if entries[0].Name != "订单库" || entries[0].Driver != "mysql8" || entries[0].Host != "db.example.com" {
		t.Fatalf("名称关联失败: %#v", entries[0])
	}
	if entries[1].Keys["user"] != "postgres" || entries[1].Keys["password"] != "pg" {
		t.Fatalf("第二条不符: %#v", entries[1])
	}
	if entries[1].Name != "日志库" {
		t.Fatalf("第二条名称关联失败: %#v", entries[1])
	}
}

func TestDecryptErrors(t *testing.T) {
	// 非 JSON 明文：解密成功但解析失败（ErrBadContent）
	dir := t.TempDir()
	encryptFixture(t, dir, "this is not json but valid padding text!")
	if _, err := OpenWorkspace(dir); !errors.Is(err, ErrBadContent) {
		t.Fatalf("应报 ErrBadContent: %v", err)
	}
	// 垃圾密文：padding 校验失败（ErrNotCredentials）
	if _, err := Decrypt([]byte("garbage-not-encrypted")); !errors.Is(err, ErrNotCredentials) {
		t.Fatalf("应报 ErrNotCredentials: %v", err)
	}
}

func TestOpenWorkspaceMissing(t *testing.T) {
	_, err := OpenWorkspace(t.TempDir())
	if !errors.Is(err, ErrNotWorkspace) {
		t.Fatalf("应报 ErrNotWorkspace: %v", err)
	}
}

func TestToolSelectAndRefresh(t *testing.T) {
	dir := t.TempDir()
	encryptFixture(t, dir, `{"c1": {"#connection": {"user": "u1", "password": "p1"}}}`)

	tl := New(DefaultState())
	r := tl.Invoke("select", dir)
	if r.IsError {
		t.Fatalf("select: %s", r.Notice)
	}
	data, ok := r.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data 类型不符: %#v", r.Data)
	}
	entries, ok := data["entries"].([]CredentialEntry)
	if !ok || len(entries) != 1 || entries[0].Keys["password"] != "p1" {
		t.Fatalf("entries 不符: %#v", data["entries"])
	}
	// 目录记忆 + Actions 下拉更新
	st := tl.State().(State)
	if st.CurrentDir != dir || len(st.RecentDirs) != 1 {
		t.Fatalf("目录未记忆: %#v", st)
	}
	// refresh 重解密
	r = tl.Invoke("refresh", "")
	if r.IsError {
		t.Fatalf("refresh: %s", r.Notice)
	}
}

func TestToolSelectError(t *testing.T) {
	tl := New(DefaultState())
	if r := tl.Invoke("select", t.TempDir()); !r.IsError {
		t.Fatal("无效目录应报错且不记忆")
	}
	if st := tl.State().(State); len(st.RecentDirs) != 0 {
		t.Fatalf("失败目录不应记忆: %#v", st)
	}
	if r := tl.Invoke("refresh", ""); !r.IsError {
		t.Fatal("未选目录 refresh 应报错")
	}
}

func TestPushRecent(t *testing.T) {
	s := DefaultState()
	for _, d := range []string{"a", "b", "c", "a"} {
		s.pushRecent(d)
	}
	if s.RecentDirs[0] != "a" || len(s.RecentDirs) != 3 {
		t.Fatalf("去重置顶不符: %#v", s.RecentDirs)
	}
}

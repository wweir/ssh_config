package ssh_config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFile(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

var files = []string{
	"testdata/config1",
	"testdata/config2",
	"testdata/eol-comments",
}

func TestDecode(t *testing.T) {
	for _, filename := range files {
		data := loadFile(t, filename)
		cfg, err := Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		out := cfg.String()
		if out != string(data) {
			t.Errorf("%s out != data: got:\n%s\nwant:\n%s\n", filename, out, string(data))
		}
	}
}

func testConfigFinder(filename string) func() string {
	return func() string { return filename }
}

func nullConfigFinder() string {
	return ""
}

func TestGet(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val := us.Get("wap", "User")
	if val != "root" {
		t.Errorf("expected to find User root, got %q", val)
	}
}

func TestJumpserverUser(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val := us.Get("jumpserver", "User")
	if val != "somebody#root#7e95b740-27dc-46fa-b18b-f71dac6e9987" {
		t.Errorf("expected to find User root, got %q", val)
	}
}

func TestGetWithDefault(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val, err := us.GetStrict("wap", "PasswordAuthentication")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if val != "yes" {
		t.Errorf("expected to get PasswordAuthentication yes, got %q", val)
	}
}

func TestGetAllWithDefault(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val, err := us.GetAllStrict("wap", "PasswordAuthentication")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(val) != 1 || val[0] != "yes" {
		t.Errorf("expected to get PasswordAuthentication yes, got %q", val)
	}
}

func TestGetIdentities(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/identities"),
	}

	val, err := us.GetAllStrict("hasidentity", "IdentityFile")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if len(val) != 1 || val[0] != "file1" {
		t.Errorf(`expected ["file1"], got %v`, val)
	}

	val, err = us.GetAllStrict("has2identity", "IdentityFile")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if len(val) != 2 || val[0] != "f1" || val[1] != "f2" {
		t.Errorf(`expected [\"f1\", \"f2\"], got %v`, val)
	}

	val, err = us.GetAllStrict("randomhost", "IdentityFile")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if len(val) != len(defaultProtocol2Identities) {
		// TODO: return the right values here.
		log.Printf("expected defaults, got %v", val)
	} else {
		for i, v := range defaultProtocol2Identities {
			if val[i] != v {
				t.Errorf("invalid %d in val, expected %s got %s", i, v, val[i])
			}
		}
	}

	val, err = us.GetAllStrict("protocol1", "IdentityFile")
	if err != nil {
		t.Errorf("expected nil err, got %v", err)
	}
	if len(val) != 1 || val[0] != "~/.ssh/identity" {
		t.Errorf("expected [\"~/.ssh/identity\"], got %v", val)
	}
}

func TestGetInvalidPort(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/invalid-port"),
	}

	val, err := us.GetStrict("test.test", "Port")
	if err == nil {
		t.Fatalf("expected non-nil err, got nil")
	}
	if val != "" {
		t.Errorf("expected to get '' for val, got %q", val)
	}
	if err.Error() != `ssh_config: strconv.ParseUint: parsing "notanumber": invalid syntax` {
		t.Errorf("wrong error: got %v", err)
	}
}

func TestGetNotFoundNoDefault(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val, err := us.GetStrict("wap", "CanonicalDomains")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if val != "" {
		t.Errorf("expected to get CanonicalDomains '', got %q", val)
	}
}

func TestGetAllNotFoundNoDefault(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val, err := us.GetAllStrict("wap", "CanonicalDomains")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(val) != 0 {
		t.Errorf("expected to get CanonicalDomains '', got %q", val)
	}
}

func TestSetDefault(t *testing.T) {
	defaultIdentity := Default("IdentityFile")
	if defaultIdentity != "~/.ssh/identity" {
		t.Fatalf("expected '~/.ssh/identity', got '%v'", defaultIdentity)
	}
	defer SetDefault("IdentityFile", defaultIdentity)

	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}
	val, err := us.GetStrict("wap", "IdentityFile")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if val != "~/.ssh/identity" {
		t.Errorf("expected to get '~/.ssh/identity', got %q", val)
	}

	SetDefault("IdentityFile", "")

	val, err = us.GetStrict("wap", "IdentityFile")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if val != "" {
		t.Errorf("expected to get empty string, got %q", val)
	}

	vals, err := us.GetAllStrict("wap", "IdentityFile")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected to get empty array, got %q", vals)
	}
}

func TestGetWildcard(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config3"),
	}

	val := us.Get("bastion.stage.i.us.example.net", "Port")
	if val != "22" {
		t.Errorf("expected to find Port 22, got %q", val)
	}

	val = us.Get("bastion.net", "Port")
	if val != "25" {
		t.Errorf("expected to find Port 24, got %q", val)
	}

	val = us.Get("10.2.3.4", "Port")
	if val != "23" {
		t.Errorf("expected to find Port 23, got %q", val)
	}
	val = us.Get("101.2.3.4", "Port")
	if val != "25" {
		t.Errorf("expected to find Port 24, got %q", val)
	}
	val = us.Get("20.20.20.4", "Port")
	if val != "24" {
		t.Errorf("expected to find Port 24, got %q", val)
	}
	val = us.Get("20.20.20.20", "Port")
	if val != "25" {
		t.Errorf("expected to find Port 25, got %q", val)
	}
}

func TestGetExtraSpaces(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/extraspace"),
	}

	val := us.Get("test.test", "Port")
	if val != "1234" {
		t.Errorf("expected to find Port 1234, got %q", val)
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config1"),
	}

	val := us.Get("wap", "uSER")
	if val != "root" {
		t.Errorf("expected to find User root, got %q", val)
	}
}

func TestGetEmpty(t *testing.T) {
	us := &UserSettings{
		userConfigFinder:   nullConfigFinder,
		systemConfigFinder: nullConfigFinder,
	}
	val, err := us.GetStrict("wap", "User")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if val != "" {
		t.Errorf("expected to get empty string, got %q", val)
	}
}

func TestGetEqsign(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/eqsign"),
	}

	val := us.Get("test.test", "Port")
	if val != "1234" {
		t.Errorf("expected to find Port 1234, got %q", val)
	}
	val = us.Get("test.test", "Port2")
	if val != "5678" {
		t.Errorf("expected to find Port2 5678, got %q", val)
	}
}

var includeFile = []byte(`
# This host should not exist, so we can use it for test purposes / it won't
# interfere with any other configurations.
Host kevinburke.ssh_config.test.example.com
    Port 4567
`)

func TestInclude(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fs write in short mode")
	}
	testPath := filepath.Join(homedir(), ".ssh", "kevinburke-ssh-config-test-file")
	err := os.WriteFile(testPath, includeFile, 0o644)
	if err != nil {
		t.Skipf("couldn't write SSH config file: %v", err.Error())
	}
	defer os.Remove(testPath)
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/include"),
	}
	val := us.Get("kevinburke.ssh_config.test.example.com", "Port")
	if val != "4567" {
		t.Errorf("expected to find Port=4567 in included file, got %q", val)
	}
}

func TestIncludeSystem(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fs write in short mode")
	}
	testPath := filepath.Join("/", "etc", "ssh", "kevinburke-ssh-config-test-file")
	err := os.WriteFile(testPath, includeFile, 0o644)
	if err != nil {
		t.Skipf("couldn't write SSH config file: %v", err.Error())
	}
	defer os.Remove(testPath)
	us := &UserSettings{
		systemConfigFinder: testConfigFinder("testdata/include"),
	}
	val := us.Get("kevinburke.ssh_config.test.example.com", "Port")
	if val != "4567" {
		t.Errorf("expected to find Port=4567 in included file, got %q", val)
	}
}

var recursiveIncludeFile = []byte(`
Host kevinburke.ssh_config.test.example.com
	Include kevinburke-ssh-config-recursive-include
`)

func TestIncludeRecursive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fs write in short mode")
	}
	testPath := filepath.Join(homedir(), ".ssh", "kevinburke-ssh-config-recursive-include")
	err := os.WriteFile(testPath, recursiveIncludeFile, 0o644)
	if err != nil {
		t.Skipf("couldn't write SSH config file: %v", err.Error())
	}
	defer os.Remove(testPath)
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/include-recursive"),
	}
	val, err := us.GetStrict("kevinburke.ssh_config.test.example.com", "Port")
	if err != ErrDepthExceeded {
		t.Errorf("Recursive include: expected ErrDepthExceeded, got %v", err)
	}
	if val != "" {
		t.Errorf("non-empty string value %s", val)
	}
}

func TestIncludeString(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fs write in short mode")
	}
	data, err := os.ReadFile("testdata/include")
	if err != nil {
		log.Fatal(err)
	}
	c, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	s := c.String()
	if s != string(data) {
		t.Errorf("mismatch: got %q\nwant %q", s, string(data))
	}
}

var matchTests = []struct {
	in    []string
	alias string
	want  bool
}{
	{[]string{"*"}, "any.test", true},
	{[]string{"a", "b", "*", "c"}, "any.test", true},
	{[]string{"a", "b", "c"}, "any.test", false},
	{[]string{"any.test"}, "any1test", false},
	{[]string{"192.168.0.?"}, "192.168.0.1", true},
	{[]string{"192.168.0.?"}, "192.168.0.10", false},
	{[]string{"*.co.uk"}, "bbc.co.uk", true},
	{[]string{"*.co.uk"}, "subdomain.bbc.co.uk", true},
	{[]string{"*.*.co.uk"}, "bbc.co.uk", false},
	{[]string{"*.*.co.uk"}, "subdomain.bbc.co.uk", true},
	{[]string{"*.example.com", "!*.dialup.example.com", "foo.dialup.example.com"}, "foo.dialup.example.com", false},
	{[]string{"test.*", "!test.host"}, "test.host", false},
}

func TestMatches(t *testing.T) {
	for _, tt := range matchTests {
		patterns := make([]*Pattern, len(tt.in))
		for i := range tt.in {
			pat, err := NewPattern(tt.in[i])
			if err != nil {
				t.Fatalf("error compiling pattern %s: %v", tt.in[i], err)
			}
			patterns[i] = pat
		}
		host := &Host{
			Patterns: patterns,
		}
		got := host.Matches(tt.alias)
		if got != tt.want {
			t.Errorf("host(%q).Matches(%q): got %v, want %v", tt.in, tt.alias, got, tt.want)
		}
	}
}

func TestMatchUnsupported(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/match-directive"),
	}

	_, err := us.GetStrict("test.test", "Port")
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexInRange(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/config4"),
	}

	user, err := us.GetStrict("wap", "User")
	if err != nil {
		t.Fatal(err)
	}
	if user != "root" {
		t.Errorf("expected User to be %q, got %q", "root", user)
	}
}

func TestDosLinesEndingsDecode(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/dos-lines"),
	}

	user, err := us.GetStrict("wap", "User")
	if err != nil {
		t.Fatal(err)
	}

	if user != "root" {
		t.Errorf("expected User to be %q, got %q", "root", user)
	}

	host, err := us.GetStrict("wap2", "HostName")
	if err != nil {
		t.Fatal(err)
	}

	if host != "8.8.8.8" {
		t.Errorf("expected HostName to be %q, got %q", "8.8.8.8", host)
	}
}

func TestNoTrailingNewline(t *testing.T) {
	us := &UserSettings{
		userConfigFinder:   testConfigFinder("testdata/config-no-ending-newline"),
		systemConfigFinder: nullConfigFinder,
	}

	port, err := us.GetStrict("example", "Port")
	if err != nil {
		t.Fatal(err)
	}

	if port != "4242" {
		t.Errorf("wrong port: got %q want 4242", port)
	}
}

func TestCustomFinder(t *testing.T) {
	us := &UserSettings{}
	us.ConfigFinder(func() string {
		return "testdata/config1"
	})

	val := us.Get("wap", "User")
	if val != "root" {
		t.Errorf("expected to find User root, got %q", val)
	}
}

func TestExtendedConfig(t *testing.T) {
	us := &UserSettings{
		userConfigFinder: testConfigFinder("testdata/exconfig"),
	}

	for _, kv := range []struct {
		host  string
		key   string
		value string
	}{
		{"ext", "User", "root"},
		{"ext", "Admin", ""},
		{"ext", "EnableTrzsz", "Yes"},
		{"ext", "EnableDragFile", "No"},
		{"ext", "Comment", ""},
		{"ext", "Password", ""},
		{"ext", "Passphrase", ""},
		{"ext2", "HostName", "::1"},
		{"ext2", "EnableTrzsz", ""},
		{"ext2", "EnableDragFile", ""},
		{"ext2", "Password", "123456"},
		{"ext2", "Passphrase", ""},
	} {
		value, err := us.GetStrict(kv.host, kv.key)
		if err != nil {
			t.Fatal(err)
		}
		if value != kv.value {
			t.Errorf("expected %q.%q to be %q, got %q", kv.host, kv.key, kv.value, value)
		}
	}
}

func TestCommentValue(t *testing.T) {
	us := &UserSettings{}
	us.ConfigFinder(func() string {
		return "testdata/comment-value"
	})

	testCases := []struct {
		key           string
		expectedValue string
	}{
		{"Key1", "Value1 # This is part of the value, not a comment"},
		{"Key2", "Value2"},
	}
	for _, tc := range testCases {
		val := us.Get("comment", tc.key)
		if val != tc.expectedValue {
			t.Errorf("expected to get %q, got %q", tc.expectedValue, val)
		}
	}
}

func TestQuoteAndSpace(t *testing.T) {
	quotePath, err := filepath.Abs("testdata/quote")
	if err != nil {
		t.Fatal(err)
	}
	spacePath, err := filepath.Abs("testdata/path with space")
	if err != nil {
		t.Fatal(err)
	}
	cfgFile := fmt.Sprintf(`Include '%s' '%s'
Host admin
  XAuthLocation /usr/bin/xauth
  IdentityAgent ~/Library/Group Containers/x x x/agent.sock
  UserKnownHostsFile "~/.ssh/kh1" ~/.ssh/kh2 "~/.ssh/k h 3"
  SetEnv LC_A=1 LC_B="2 3" "LC_C=4 5 6"
  SetEnv = 'LC_D = 7 "8 9"'
`, quotePath, spacePath)

	cfg, err := Decode(bytes.NewReader([]byte(cfgFile)))
	if err != nil {
		t.Fatal(err)
	}

	for _, kv := range []struct {
		host  string
		key   string
		value string
	}{
		{"admin", "XAuthLocation", "/usr/bin/xauth"},
		{"quote", "XAuthLocation", "/usr/bin/xauth"},
		{"space", "XAuthLocation", "/usr/bin/xauth"},
		{"admin", "IdentityAgent", "~/Library/Group Containers/x x x/agent.sock"},
		{"quote", "IdentityAgent", "~/Library/Group Containers/x x x/agent.sock"},
		{"space", "IdentityAgent", "~/Library/Group Containers/x x x/agent.sock"},
		{"admin", "UserKnownHostsFile", `"~/.ssh/kh1" ~/.ssh/kh2 "~/.ssh/k h 3"`},
		{"quote", "UserKnownHostsFile", `~/.ssh/kh1 ~/.ssh/kh2 "~/.ssh/k h 3"`},
		{"space", "UserKnownHostsFile", `'~/.ssh/kh1' ~/.ssh/kh2 "~/.ssh/k h 3"`},
	} {
		value, err := cfg.Get(kv.host, kv.key)
		if err != nil {
			t.Fatal(err)
		}
		if value != kv.value {
			t.Errorf("expected %q.%q to be %q, got %q", kv.host, kv.key, kv.value, value)
		}
	}

	for _, kv := range []struct {
		host   string
		key    string
		values []string
	}{
		{"admin", "UserKnownHostsFile", []string{"~/.ssh/kh1", "~/.ssh/kh2", "~/.ssh/k h 3"}},
		{"quote", "UserKnownHostsFile", []string{"~/.ssh/kh1", "~/.ssh/kh2", "~/.ssh/k h 3"}},
		{"space", "UserKnownHostsFile", []string{"~/.ssh/kh1", "~/.ssh/kh2", "~/.ssh/k h 3"}},
	} {
		values, err := cfg.GetSplits(kv.host, kv.key)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(values, kv.values) {
			t.Errorf("expected %q.%q to be %q, got %q", kv.host, kv.key, kv.values, values)
		}
	}

	for _, kv := range []struct {
		host   string
		key    string
		values []string
	}{
		{"admin", "SetEnv", []string{`LC_A=1 LC_B="2 3" "LC_C=4 5 6"`, `LC_D = 7 "8 9"`}},
		{"quote", "SetEnv", []string{`'LC_A=1' LC_B='2 3' 'LC_C=4 5 6'`, `LC_D = 7 "8 9"`}},
		{"space", "SetEnv", []string{`"LC_A=1" LC_B="2 3" LC_C="4 5 6"`, `LC_D = 7 "8 9"`}},
	} {
		values, err := cfg.GetAll(kv.host, kv.key)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(values, kv.values) {
			t.Errorf("expected %q.%q to be %q, got %q", kv.host, kv.key, kv.values, values)
		}
	}

	for _, kv := range []struct {
		host   string
		key    string
		values []string
	}{
		{"admin", "SetEnv", []string{"LC_A=1", "LC_B=2 3", "LC_C=4 5 6", `LC_D = 7 "8 9"`}},
		{"quote", "SetEnv", []string{"LC_A=1", "LC_B=2 3", "LC_C=4 5 6", `LC_D = 7 "8 9"`}},
		{"space", "SetEnv", []string{"LC_A=1", "LC_B=2 3", "LC_C=4 5 6", `LC_D = 7 "8 9"`}},
	} {
		values, err := cfg.GetAllSplits(kv.host, kv.key)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(values, kv.values) {
			t.Errorf("expected %q.%q to be %q, got %q", kv.host, kv.key, kv.values, values)
		}
	}
}

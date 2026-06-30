package ostruct

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// rubyInspect runs an `ruby -rostruct` snippet that must `print` (no newline)
// its result, and returns the output. The test is skipped when ruby is absent,
// on Windows (no POSIX ruby in CI), or when the available ruby is not the
// MRI 4.x line this core tracks. Output is read raw; we never depend on a
// trailing newline (binmode-safe across platforms).
func rubyInspect(t *testing.T, snippet string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("differential test skipped on windows")
	}
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not found; skipping differential test")
	}
	ver, err := exec.Command(path, "-e", "print RUBY_VERSION").Output()
	if err != nil {
		t.Skipf("ruby --version failed: %v", err)
	}
	if !strings.HasPrefix(string(ver), "4.") {
		t.Skipf("ruby %s is not the tracked 4.x line", strings.TrimSpace(string(ver)))
	}
	out, err := exec.Command(path, "-rostruct", "-e", snippet).Output()
	if err != nil {
		t.Fatalf("ruby snippet failed: %v\nsnippet: %s", err, snippet)
	}
	return strings.TrimRight(string(out), "\r\n")
}

func TestDiffInspect(t *testing.T) {
	cases := []struct {
		os      *OpenStruct
		snippet string
	}{
		{New(), `print OpenStruct.new.inspect`},
		{
			New(Pair{"name", "John"}, Pair{"age", 70}),
			`print OpenStruct.new(name: "John", age: 70).inspect`,
		},
		{
			New(Pair{"s", `x"y`}, Pair{"n", nil}, Pair{"sym", Symbol("foo")}, Pair{"arr", []any{1, 2}}),
			`print OpenStruct.new(s: "x\"y", n: nil, sym: :foo, arr: [1, 2]).inspect`,
		},
		{
			New(Pair{"f", 1.5}, Pair{"t", true}, Pair{"fa", false}),
			`print OpenStruct.new(f: 1.5, t: true, fa: false).inspect`,
		},
	}
	for _, c := range cases {
		want := rubyInspect(t, c.snippet)
		if got := c.os.Inspect(); got != want {
			t.Errorf("inspect mismatch\n go:   %q\n ruby: %q", got, want)
		}
	}
}

func TestDiffToHOrder(t *testing.T) {
	want := rubyInspect(t, `print OpenStruct.new(b: 1, a: 2, c: 3).to_h.keys.map(&:to_s).join(",")`)
	o := New(Pair{"b", 1}, Pair{"a", 2}, Pair{"c", 3})
	var keys []string
	for _, p := range o.ToH() {
		keys = append(keys, string(p.Key.(Symbol)))
	}
	if got := strings.Join(keys, ","); got != want {
		t.Errorf("to_h order: go %q, ruby %q", got, want)
	}
}

func TestDiffDig(t *testing.T) {
	want := rubyInspect(t, `print OpenStruct.new(a: {b: {c: 1}}).dig(:a, :b, :c)`)
	// model the nested ruby Hash with nested OpenStructs (dig delegates the same)
	o := New(Pair{"a", New(Pair{"b", New(Pair{"c", 1})})})
	v, err := o.Dig(Symbol("a"), Symbol("b"), Symbol("c"))
	if err != nil {
		t.Fatal(err)
	}
	if got := InspectValue(v); got != want {
		t.Errorf("dig: go %q, ruby %q", got, want)
	}
}

func TestDiffDeleteField(t *testing.T) {
	// returns old value
	want := rubyInspect(t, `print OpenStruct.new(name: "John", age: 70).delete_field(:age)`)
	o := New(Pair{"name", "John"}, Pair{"age", 70})
	v, err := o.DeleteField(Symbol("age"))
	if err != nil {
		t.Fatal(err)
	}
	if got := InspectValue(v); got != want {
		t.Errorf("delete_field value: go %q, ruby %q", got, want)
	}
	// to_h after delete
	wantH := rubyInspect(t, `o = OpenStruct.new(name: "John", age: 70); o.delete_field(:age); print o.to_h.map{|k,v| "#{k}=#{v.inspect}"}.join(",")`)
	var parts []string
	for _, p := range o.ToH() {
		parts = append(parts, string(p.Key.(Symbol))+"="+InspectValue(p.Value))
	}
	if got := strings.Join(parts, ","); got != wantH {
		t.Errorf("to_h after delete: go %q, ruby %q", got, wantH)
	}
}

func TestDiffDeleteFieldAbsentMessage(t *testing.T) {
	want := rubyInspect(t, `begin; OpenStruct.new.delete_field(:x); rescue NameError => e; print e.message; end`)
	o := New()
	_, err := o.DeleteField(Symbol("x"))
	if err == nil || err.Error() != want {
		t.Errorf("delete_field absent: go %v, ruby %q", err, want)
	}
}

func TestDiffEqual(t *testing.T) {
	want := rubyInspect(t, `print(OpenStruct.new(name: "John") == OpenStruct.new(name: "John"))`)
	got := "false"
	if New(Pair{"name", "John"}).Equal(New(Pair{"name", "John"})) {
		got = "true"
	}
	if got != want {
		t.Errorf("==: go %q, ruby %q", got, want)
	}

	want2 := rubyInspect(t, `print(OpenStruct.new(a: 1) == OpenStruct.new(a: 1, b: 2))`)
	got2 := "false"
	if New(Pair{"a", 1}).Equal(New(Pair{"a", 1}, Pair{"b", 2})) {
		got2 = "true"
	}
	if got2 != want2 {
		t.Errorf("!=: go %q, ruby %q", got2, want2)
	}
}

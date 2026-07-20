package printer

import "testing"

func TestEdgeCases(t *testing.T) {
	src := `f = @(x, y) x + y*2;
g = @sin;
s.field = 1;
s.(dynname) = 2;
c{1} = 'x';
r = 1:2:10;
do
  x = x + 1;
until x > 10
parfor i = 1:10, 4
  y(i) = i^2;
endparfor
global a b = 1
persistent count
unwind_protect
  risky();
unwind_protect_cleanup
  cleanup();
end
n = ~flag;
m = a.'  ;
p = a' * b;
w = -x^2;
q = (a+b)*c;
`
	out := format(t, src)
	t.Log("\n" + out)
	out2 := format(t, out)
	if out != out2 {
		t.Fatalf("not idempotent:\n---1---\n%s\n---2---\n%s", out, out2)
	}
}

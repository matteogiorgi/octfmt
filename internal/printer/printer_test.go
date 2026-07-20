package printer

import (
	"testing"

	"github.com/matteo/octave-formatter/internal/parser"
)

func format(t *testing.T, src string) string {
	t.Helper()
	f, errs := parser.Parse(src)
	for _, e := range errs {
		t.Fatalf("parse error: %v", e)
	}
	return Print(f, DefaultOptions)
}

func TestFormatBasic(t *testing.T) {
	src := `
function   y=square( x )
% squares x
y=x^2;
endfunction



a=[1 -1 2,3;4 5 6]';


b={1,'two',"three"};
if a>0&&b<1
disp('positive')
elseif a==0
disp 'zero'
else
disp('negative')
end

for i=1:10
s=s+i;
end



[m,n]=size(a);
[~,idx]=max(a);
`
	out := format(t, src)
	t.Log("\n" + out)
}

func TestIdempotent(t *testing.T) {
	src := `function y = square(x)
  % squares x
  y = x^2;
endfunction

a = [1, -1, 2, 3
     4, 5, 6]';
b = {1, 'two', "three"};
if a > 0 && b < 1
  disp('positive')
elseif a == 0
  disp 'zero'
else
  disp('negative')
end

for i = 1:10
  s = s + i;
end

[m, n] = size(a);
[~, idx] = max(a);

switch x
  case 1
    y = 1;

  case {2, 3}
    y = 2;
  otherwise
    y = 0;
end

try
  risky();
catch err
  disp(err.message)
end
`
	out1 := format(t, src)
	out2 := format(t, out1)
	if out1 != out2 {
		t.Fatalf("not idempotent:\n--- first ---\n%s\n--- second ---\n%s", out1, out2)
	}
}

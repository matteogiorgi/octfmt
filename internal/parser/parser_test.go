package parser

import "testing"

func TestSmoke(t *testing.T) {
	src := `
function y = square(x)
  % squares x
  y = x^2;
endfunction

a = [1 -1 2, 3; 4 5 6]';
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
	f, errs := Parse(src)
	for _, e := range errs {
		t.Errorf("parse error: %v", e)
	}
	if f == nil {
		t.Fatal("nil file")
	}
	if len(f.Stmts) == 0 {
		t.Fatal("no statements parsed")
	}
	t.Logf("parsed %d top-level statements", len(f.Stmts))
}

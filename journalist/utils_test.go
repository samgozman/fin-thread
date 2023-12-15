package journalist

import (
	"testing"
)

func Test_replaceUnicodeSymbols(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"ampersand test",
			args{"S\\u0026P 500"},
			"S&P 500",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replaceUnicodeSymbols(tt.args.s); got != tt.want {
				t.Errorf("replaceHTMLCodeSymbols() = %v, want %v", got, tt.want)
			}
		})
	}
}

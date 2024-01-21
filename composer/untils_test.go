package composer

import "testing"

func Test_aiJSONStringFixer(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test with empty array",
			args: args{
				str: "[[]]",
			},
			want:    "[]",
			wantErr: false,
		},
		{
			name: "Test with array containing backslash and hallucinations text",
			args: args{
				str: "some meh \n [\\] \n some blah",
			},
			want:    "[]",
			wantErr: false,
		},
		{
			name: "Test with array group (should be the first group)",
			args: args{
				str: "some array [[{\"a\": 1}]]",
			},
			want:    "[{\"a\": 1}]",
			wantErr: false,
		},
		{
			name: "Test with no array",
			args: args{
				str: "no array",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := aiJSONStringFixer(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("aiJSONStringFixer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("aiJSONStringFixer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

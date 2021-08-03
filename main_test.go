package main

import (
	"net/url"
	"testing"
)

func Test_collectWatsons(t *testing.T) {
	type args struct {
		prodname string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "a normal page",
			args: args{
				prodname: url.QueryEscape("指甲"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collectWatsons(tt.args.prodname); (err != nil) != tt.wantErr {
				t.Errorf("collectWatsons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package main

import (
	"net/url"
	"testing"
)

func Test_collectWatsons(t *testing.T) {
	testf := true
	type args struct {
		prodname string
		flag     *bool
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
				flag:     &testf,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collectWatsons(tt.args.prodname, tt.args.flag); (err != nil) != tt.wantErr {
				t.Errorf("collectWatsons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_collectEbay(t *testing.T) {
	testf := true
	type args struct {
		prodname string
		flag     *bool
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
				prodname: url.QueryEscape("hello"),
				flag:     &testf,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := collectWatsons(tt.args.prodname, tt.args.flag); (err != nil) != tt.wantErr {
				t.Errorf("collectWatsons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package crawl

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestSearchWeb(t *testing.T) {
	type args struct {
		ctx      context.Context
		prodName string
		w        http.ResponseWriter
		r        *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    *[]string
		wantErr bool
	}{

		{
			name: "test1",
			args: args{

				prodName: "100",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchWeb(tt.args.ctx, tt.args.prodName, tt.args.w, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchWeb() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SearchWeb() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Task Board is a self-hosted Kanban application.
// Copyright 2026-present Task Board contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package utils

import "testing"

func TestMd5String(t *testing.T) {
	type args struct {
		cleartext string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test md5 generation",
			args: args{cleartext: "taskboardrandomstringwhichisnotrandom"},
			want: "10da997aad0311fd6fbe4c1215c4084f",
		},
		{
			name: "Test md5 generation",
			args: args{cleartext: "taskboardstring"},
			want: "5d530aa6359147fc9da8b7232e863c6c",
		},
		{
			name: "Test md5 generation",
			args: args{cleartext: "somethingsomething"},
			want: "2264cdc4cf48a80cc00d23730b6c03ea",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Md5String(tt.args.cleartext); got != tt.want {
				t.Errorf("Md5String() = %v, want %v", got, tt.want)
			}
		})
	}
}

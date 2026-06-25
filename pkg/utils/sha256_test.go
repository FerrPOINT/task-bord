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

func TestSha256(t *testing.T) {
	type args struct {
		cleartext string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test sha256 generation",
			args: args{cleartext: "taskboardrandomstringwhichisnotrandom"},
			want: "49cd7b9bd18d9eabb81fcb10811a1686fccd8f1843ce9",
		},
		{
			name: "Test sha256 generation",
			args: args{cleartext: "taskboardstring"},
			want: "f27ec4529824bef51efdf49f023a3995e54a3f5db8fc4",
		},
		{
			name: "Test sha256 generation",
			args: args{cleartext: "somethingsomething"},
			want: "00aef67d6df7fdee0419aa3713820e7084cbcb8b8f7c4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sha256(tt.args.cleartext); got != tt.want {
				t.Errorf("Sha256() = %v, want %v", got, tt.want)
			}
		})
	}
}

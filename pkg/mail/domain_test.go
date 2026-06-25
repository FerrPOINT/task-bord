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

package mail

import (
	"os"
	"testing"

	"github.com/FerrPOINT/task-bord/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGetMailDomain(t *testing.T) {
	t.Run("falls back to os.Hostname when public URL is empty", func(t *testing.T) {
		config.ServicePublicURL.Set("")
		expectedHostname, err := os.Hostname()
		if err != nil || expectedHostname == "" {
			assert.Equal(t, "taskboard", GetMailDomain())
		} else {
			assert.Equal(t, expectedHostname, GetMailDomain())
		}
	})

	t.Run("extracts hostname from public URL", func(t *testing.T) {
		config.ServicePublicURL.Set("https://tasks.example.com/")
		assert.Equal(t, "tasks.example.com", GetMailDomain())
	})

	t.Run("extracts hostname without port", func(t *testing.T) {
		config.ServicePublicURL.Set("https://tasks.example.com:8080/")
		assert.Equal(t, "tasks.example.com", GetMailDomain())
	})

	t.Run("falls back to os.Hostname for invalid URL", func(t *testing.T) {
		config.ServicePublicURL.Set("://bad")
		expectedHostname, err := os.Hostname()
		if err != nil || expectedHostname == "" {
			assert.Equal(t, "taskboard", GetMailDomain())
		} else {
			assert.Equal(t, expectedHostname, GetMailDomain())
		}
	})
}

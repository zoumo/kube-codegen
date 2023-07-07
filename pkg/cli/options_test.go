// Copyright 2022 jim.zoumo@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"io/fs"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_goFileExists(t *testing.T) {
	memFS := afero.NewMemMapFs()
	memFS.MkdirAll("pkg/apis/apps/v1", fs.ModePerm)
	memFS.MkdirAll("pkg/apis/apps/v2", fs.ModePerm)
	memFS.Create("pkg/apis/apps/types.go")
	memFS.Create("pkg/apis/apps/v2/types.go")
	iofs := afero.NewIOFS(memFS)

	got, _ := goFileExists(iofs, "pkg/apis/apps")
	want := true
	assert.Equal(t, want, got)

	got, _ = goFileExists(iofs, "pkg/apis/apps/v1")
	want = false
	assert.Equal(t, want, got)

	got, _ = goFileExists(iofs, "pkg/apis/apps/v2")
	want = true
	assert.Equal(t, want, got)
}

func Test_findGroupVersion(t *testing.T) {
	memFS := afero.NewMemMapFs()
	memFS.MkdirAll("pkg/apis/apps/v1", fs.ModePerm)
	memFS.MkdirAll("pkg/apis/apps/v2", fs.ModePerm)
	memFS.Create("pkg/apis/apps/types.go")
	memFS.Create("pkg/apis/apps/v2/types.go")
	iofs := afero.NewIOFS(memFS)

	groupVersions, internalGroupVersions, _ := findGroupVersion(iofs, "pkg/apis")

	assert.Equal(t, []string{"apps/v1", "apps/v2"}, groupVersions)
	assert.Equal(t, []string{"apps"}, internalGroupVersions)
}

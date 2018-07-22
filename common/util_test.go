package common

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevNull_Write(t *testing.T) {
	var writer DevNull
	bytes := []byte("hello world")
	n, err := writer.Write(bytes)
	assert.Nil(t, err)
	assert.Equal(t, len(bytes), n)
}

func TestGetFullPath(t *testing.T) {
	_, err := GetFullPath("inertia.go")
	assert.Nil(t, err)
}

func TestGenerateRandomString(t *testing.T) {
	randString, err := GenerateRandomString()
	assert.Nil(t, err)
	assert.Equal(t, len(randString), 44)
}

func TestCheckForDockerCompose(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)

	ymlPath := path.Join(cwd, "/docker-compose.yml")

	// No!
	b := CheckForDockerCompose(cwd)
	assert.False(t, b)

	// Yes!
	file, err := os.Create(ymlPath)
	assert.Nil(t, err)
	file.Close()
	b = CheckForDockerCompose(cwd)
	assert.True(t, b)
	os.Remove(ymlPath)
}

func TestCheckForDockerfile(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)

	path := path.Join(cwd, "/Dockerfile")

	// No!
	b := CheckForDockerfile(cwd)
	assert.False(t, b)

	// Yes!
	file, err := os.Create(path)
	assert.Nil(t, err)
	file.Close()
	b = CheckForDockerfile(cwd)
	assert.True(t, b)
	os.Remove(path)
}

func TestCheckForProcfile(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)

	path := path.Join(cwd, "/Procfile")

	// No!
	b := CheckForProcfile(cwd)
	assert.False(t, b)

	// Yes!
	file, err := os.Create(path)
	assert.Nil(t, err)
	file.Close()
	b = CheckForProcfile(cwd)
	assert.True(t, b)
	os.Remove(path)
}

func TestExtract(t *testing.T) {
	for _, url := range remoteURLVariations {
		repoName := ExtractRepository(url)
		assert.Equal(t, "ubclaunchpad/inertia", repoName)
	}

	repoNameWithHyphens := ExtractRepository("git@github.com:ubclaunchpad/inertia-deploy-test.git")
	assert.Equal(t, "ubclaunchpad/inertia-deploy-test", repoNameWithHyphens)

	repoNameWithDots := ExtractRepository("git@github.com:ubclaunchpad/inertia.deploy.test.git")
	assert.Equal(t, "ubclaunchpad/inertia.deploy.test", repoNameWithDots)

	repoNameWithMixed := ExtractRepository("git@github.com:ubclaunchpad/inertia-deploy.test.git")
	assert.Equal(t, "ubclaunchpad/inertia-deploy.test", repoNameWithMixed)
}

func TestRemoveContents(t *testing.T) {
	testdir := filepath.Join(".", "test")
	type args struct {
		directory  string
		removeTOML bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"not a directory", args{"", true}, true},
		{"remove all", args{testdir, true}, false},
		{"keep toml", args{testdir, false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create directory and remove on exit
			err := os.Mkdir(testdir, os.ModePerm)
			assert.Nil(t, err)
			defer os.RemoveAll(testdir)

			// Set up files
			tomlPath := filepath.Join("./test", "inertia.toml")
			f, err := os.Create(tomlPath)
			assert.Nil(t, err)
			f.Close()
			f, err = os.Create(filepath.Join("./test", "somefile"))
			assert.Nil(t, err)
			f.Close()

			// Run test
			if err := RemoveContents(tt.args.directory, tt.args.removeTOML); (err != nil) != tt.wantErr {
				t.Errorf("RemoveContents() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if tt.args.removeTOML {
					empty, err := isDirEmpty(testdir)
					assert.Nil(t, err)
					assert.True(t, empty, "Should be empty")
				} else {
					_, err := os.Stat(tomlPath)
					assert.Nil(t, err, "inertia.toml should exist")
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	assert.NotNil(t, ParseDate("2006-01-02T15:04:05.000Z"))
}

func TestParseInt64(t *testing.T) {
	parsed, err := ParseInt64("10")
	assert.Nil(t, err)
	assert.Equal(t, int64(10), parsed)

	_, err = ParseInt64("")
	assert.NotNil(t, err)
}

package config

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		expected *Config
		err      bool
		filename string
		mock     func()
		cleanup  func() error
	}{
		{
			name: "Successful config",
			expected: &Config{
				Host: "localhost",
				Port: "8080",
				Redis: struct {
					Host string "json:\"host\""
					Port string "json:\"port\""
				}{
					Host: "127.0.0.1",
					Port: "6379",
				},
			},
			filename: "test.json",
			mock: func() {
				file, _ := os.Create("test.json")
				defer file.Close()

				io.WriteString(file, `{"host": "localhost","port":"8080","redis":{"host":"127.0.0.1", "port":"6379"}}`)
			},
			cleanup: func() error {
				return os.Remove("test.json")
			},
		},
		{
			name:     "File doesn't exist",
			expected: nil,
			err:      true,
			filename: "test2.json",
			mock:     func() {},
			cleanup:  func() error { return nil },
		},
		{
			name:     "Badly formatted JSON.",
			expected: nil,
			err:      true,
			filename: "test3.json",
			mock: func() {
				file, _ := os.Create("test3.json")
				defer file.Close()

				io.WriteString(file, `{"host": "localhost","port":"8080","redis":{"host":"127.0.0.1", "port":"6379"}`)
			},
			cleanup: func() error {
				return os.Remove("test3.json")
			},
		},
	}

	for _, test := range tests {
		test.mock()

		config, err := New(test.filename)
		if test.err {
			assert.Error(t, err, test.name)
		} else {
			assert.Equal(t, test.expected, config, test.name)
		}

		err = test.cleanup()
		assert.Nil(t, err)
	}
}

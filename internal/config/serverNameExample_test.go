package config

import (
	"testing"

	"github.com/hankyu66/sponge/configs"

	"github.com/hankyu66/sponge/pkg/gofile"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	configFile := configs.Path("serverNameExample.yml")
	err := Init(configFile)
	if gofile.IsExists(configFile) {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}

	c := Get()
	assert.NotNil(t, c)

	str := Show()
	assert.NotEmpty(t, str)

	// set nil
	Set(nil)
	defer func() {
		recover()
	}()
	Get()
}

func TestInitNacos(t *testing.T) {
	configFile := configs.Path("serverNameExample_cc.yml")
	_, err := NewCenter(configFile)
	if gofile.IsExists(configFile) {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
	}
}

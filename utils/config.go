// Ultimate Provisioner: UP cmd
// Copyright (c) 2019 Stephen Cheng and contributors

/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package utils

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path"
	"reflect"
)

var (
	MainConfig *UpConfig
)

type UpConfigLoader struct {
	Dir     string
	YmlFile string
}

func NewUpConfig(configdir, configymlfile string) *UpConfig {
	upCfg := UpConfigLoader{
		Dir:     configdir,
		YmlFile: configymlfile,
	}

	dir := func() (s string) {
		if upCfg.Dir == "" {
			s = defaults["ConfigDir"]
		} else {
			s = upCfg.Dir
		}
		return
	}()
	filename := func() (s string) {
		if upCfg.YmlFile == "" {
			s = defaults["ConfigFile"]
		} else {
			s = upCfg.YmlFile
		}
		return
	}()

	filepath := path.Join(dir, filename)
	var config *viper.Viper
	if _, err := os.Stat(filepath); err == nil {
		config = YamlLoader("Config", dir, filename)
	} else {
		LogWarn("config file does not exist", "use builtin defaults")
	}

	cfg := new(UpConfig)
	if config != nil {
		err := config.Unmarshal(cfg)
		if err != nil {
			fmt.Println("unable to decode into struct:", err.Error())
		}
	}

	return cfg
}

func (cfg *UpConfig) InitConfig() *UpConfig {
	e := reflect.ValueOf(cfg).Elem()
	et := reflect.Indirect(e).Type()

	for i := 0; i < e.NumField(); i++ {
		//currently only support string type field
		if f := e.Field(i); f.Kind() == reflect.String {
			fname := et.Field(i).Name
			if val, ok := defaults[fname]; ok {
				if f.String() == "" {
					f.SetString(val)
				}
			}
		}
	}

	if cfg.ModuleName == "" {
		cfg.ModuleName = GetRandomName(1)
	}

	cfg.SetAbsWorkdir()
	return cfg
}



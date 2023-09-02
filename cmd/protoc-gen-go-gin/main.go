// Package main is to generate *.go(tmpl), *_router.go, *_http.go, *_router.pb.go,files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zhufuyi/sponge/cmd/protoc-gen-go-gin/internal/generate/handler"
	"github.com/zhufuyi/sponge/cmd/protoc-gen-go-gin/internal/generate/router"
	"github.com/zhufuyi/sponge/cmd/protoc-gen-go-gin/internal/generate/service"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	handlerPlugin = "handler"
	servicePlugin = "service"

	helpInfo = `
# generate *_router.pb.go file
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative *.proto

# generate *_router.pb.go, *.go(tmpl), *_router.go, *_http.go files
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative --go-gin_opt=plugin=handler \
  --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName *.proto

# generate *_router.pb.go, *.go(tmpl), *_router.go, *_rpc.go files
protoc --proto_path=. --proto_path=./third_party --go-gin_out=. --go-gin_opt=paths=source_relative --go-gin_opt=plugin=service \
  --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName *.proto

Note:
    If you want to merge the code, after generating the code, execute the command "sponge merge http-pb" or
    "sponge merge rpc-gw-pb", you don't worry about it affecting the logic code you have already written,
    in case of accidents, you can find the pre-merge code in the directory /tmp/sponge_merge_backup_code.
`

	optErrFormat = `--go-gin_opt error, '%s' cannot be empty.

Usage example: 
    protoc --proto_path=. --proto_path=./third_party \
      --go-gin_out=. --go-gin_opt=paths=source_relative \
      --go-gin_opt=plugin=%s --go-gin_opt=moduleName=yourModuleName --go-gin_opt=serverName=yourServerName \
      *.proto
`
)

func main() {
	var h bool
	flag.BoolVar(&h, "h", false, "help information")
	flag.Parse()
	if h {
		fmt.Printf("%s", helpInfo)
		return
	}

	var flags flag.FlagSet

	var plugin, moduleName, serverName, logicOut, routerOut, ecodeOut string
	flags.StringVar(&plugin, "plugin", "", "plugin name, supported values: handler or service")
	flags.StringVar(&moduleName, "moduleName", "", "module name for plugin")
	flags.StringVar(&serverName, "serverName", "", "server name for plugin")
	flags.StringVar(&logicOut, "logicOut", "", "directory of logical template code generated by the plugin, "+
		"the default value is internal/handler if the plugin is a handler, or internal/service if it is a service")
	flags.StringVar(&routerOut, "routerOut", "", "directory of routing code generated by the plugin, default is internal/routers")
	flags.StringVar(&ecodeOut, "ecodeOut", "", "directory of error code generated by the plugin, default is internal/ecode")

	options := protogen.Options{
		ParamFunc: flags.Set,
	}

	options.Run(func(gen *protogen.Plugin) error {
		handlerFlag, serviceFlag := false, false
		pluginName := strings.ReplaceAll(plugin, " ", "")
		switch pluginName {
		case handlerPlugin:
			handlerFlag = true
			if logicOut == "" {
				logicOut = "internal/handler"
			}
			if routerOut == "" {
				routerOut = "internal/routers"
			}
			if ecodeOut == "" {
				ecodeOut = "internal/ecode"
			}
		case servicePlugin:
			serviceFlag = true
			if logicOut == "" {
				logicOut = "internal/service"
			}
			if routerOut == "" {
				routerOut = "internal/routers"
			}
			if ecodeOut == "" {
				ecodeOut = "internal/ecode"
			}
		case "":
		default:
			return fmt.Errorf("protoc-gen-go-gin: unknown plugin %q", plugin)
		}

		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			router.GenerateFile(gen, f)

			if handlerFlag {
				err := saveHandlerAndRouterFiles(f, moduleName, serverName, logicOut, routerOut, ecodeOut)
				if err != nil {
					continue // skip error, process the next protobuf file
				}
			} else if serviceFlag {
				err := saveServiceAndRouterFiles(f, moduleName, serverName, logicOut, routerOut, ecodeOut)
				if err != nil {
					continue // skip error, process the next protobuf file
				}
			}
		}
		return nil
	})
}

func saveHandlerAndRouterFiles(f *protogen.File, moduleName string, serverName string,
	logicOut string, routerOut string, ecodeOut string) error {
	filenamePrefix := f.GeneratedFilenamePrefix
	handlerLogicContent, routerContent, errCodeFileContent := handler.GenerateFiles(f)

	filePath := filenamePrefix + ".go"
	err := saveFile(moduleName, serverName, logicOut, filePath, handlerLogicContent, false, handlerPlugin)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_router.go"
	err = saveFile(moduleName, serverName, routerOut, filePath, routerContent, false, handlerPlugin)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_http.go"
	err = saveFileSimple(ecodeOut, filePath, errCodeFileContent, false)
	if err != nil {
		return err
	}

	return nil
}

func saveServiceAndRouterFiles(f *protogen.File, moduleName string, serverName string,
	logicOut string, routerOut string, ecodeOut string) error {
	filenamePrefix := f.GeneratedFilenamePrefix
	serviceLogicContent, routerContent, errCodeFileContent := service.GenerateFiles(f)

	filePath := filenamePrefix + ".go"
	err := saveFile(moduleName, serverName, logicOut, filePath, serviceLogicContent, false, servicePlugin)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_router.go"
	err = saveFile(moduleName, serverName, routerOut, filePath, routerContent, false, servicePlugin)
	if err != nil {
		return err
	}

	filePath = filenamePrefix + "_rpc.go"
	err = saveFileSimple(ecodeOut, filePath, errCodeFileContent, false)
	if err != nil {
		return err
	}

	return nil
}

func saveFile(moduleName string, serverName string, out string, filePath string, content []byte, isNeedCovered bool, pluginName string) error {
	if len(content) == 0 {
		return nil
	}

	if moduleName == "" {
		panic(fmt.Sprintf(optErrFormat, "moduleName", pluginName))
	}
	if serverName == "" {
		panic(fmt.Sprintf(optErrFormat, "serverName", pluginName))
	}

	_ = os.MkdirAll(out, 0766)
	_, name := filepath.Split(filePath)
	file := out + "/" + name
	if !isNeedCovered && isExists(file) {
		file += ".gen" + time.Now().Format("20060102T150405")
	}

	content = bytes.ReplaceAll(content, []byte("moduleNameExample"), []byte(moduleName))
	content = bytes.ReplaceAll(content, []byte("serverNameExample"), []byte(serverName))
	content = bytes.ReplaceAll(content, firstLetterToUpper("serverNameExample"), firstLetterToUpper(serverName))
	return os.WriteFile(file, content, 0666)
}

func saveFileSimple(out string, filePath string, content []byte, isNeedCovered bool) error {
	if len(content) == 0 {
		return nil
	}

	_ = os.MkdirAll(out, 0766)
	_, name := filepath.Split(filePath)
	file := out + "/" + name
	if !isNeedCovered && isExists(file) {
		file += ".gen" + time.Now().Format("20060102T150405")
	}

	return os.WriteFile(file, content, 0666)
}

func isExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}

func firstLetterToUpper(s string) []byte {
	if s == "" {
		return []byte{}
	}

	return []byte(strings.ToUpper(s[:1]) + s[1:])
}

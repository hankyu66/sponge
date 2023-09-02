package generate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zhufuyi/sponge/pkg/replacer"
	"github.com/zhufuyi/sponge/pkg/sql2code"
	"github.com/zhufuyi/sponge/pkg/sql2code/parser"

	"github.com/spf13/cobra"
)

// DaoCommand generate dao code
func DaoCommand(parentName string) *cobra.Command {
	var (
		moduleName      string // go.mod module name
		outPath         string // output directory
		dbTables        string // table names
		isIncludeInitDB bool

		sqlArgs = sql2code.Args{
			Package:  "model",
			JSONTag:  true,
			GormType: true,
		}
	)

	cmd := &cobra.Command{
		Use:   "dao",
		Short: "Generate dao code based on mysql table",
		Long: fmt.Sprintf(`generate dao code based on mysql table.

Examples:
  # generate dao code and embed gorm.model struct.
  sponge %s dao --module-name=yourModuleName --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user

  # generate dao code with multiple table names.
  sponge %s dao --module-name=yourModuleName --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=t1,t2

  # generate dao code, structure fields correspond to the column names of the table.
  sponge %s dao --module-name=yourModuleName --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --embed=false

  # generate dao code and specify the server directory, Note: code generation will be canceled when the latest generated file already exists.
  sponge %s dao --db-dsn=root:123456@(192.168.3.37:3306)/test --db-table=user --out=./yourServerDir
`, parentName, parentName, parentName, parentName),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			mdName, _ := getNamesFromOutDir(outPath)
			if mdName != "" {
				moduleName = mdName
			} else if moduleName == "" {
				return fmt.Errorf(`required flag(s) "module-name" not set, use "sponge %s dao -h" for help`, parentName)
			}

			tableNames := strings.Split(dbTables, ",")
			for count, tableName := range tableNames {
				if tableName == "" {
					continue
				}

				sqlArgs.DBTable = tableName
				codes, err := sql2code.Generate(&sqlArgs)
				if err != nil {
					return err
				}

				// control to generate the initialization db code only once
				if count == 0 && isIncludeInitDB {
					isIncludeInitDB = true
				} else {
					isIncludeInitDB = false
				}

				outPath, err = runGenDaoCommand(moduleName, isIncludeInitDB, codes, outPath)
				if err != nil {
					return err
				}
			}

			fmt.Printf(`
using help:
  move the folder "internal" to your project code folder.

`)
			fmt.Printf("generate \"dao\" code successfully, out = %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&moduleName, "module-name", "m", "", "module-name is the name of the module in the go.mod file")
	//_ = cmd.MarkFlagRequired("module-name")
	cmd.Flags().StringVarP(&sqlArgs.DBDsn, "db-dsn", "d", "", "db content addr, e.g. user:password@(host:port)/database")
	_ = cmd.MarkFlagRequired("db-dsn")
	cmd.Flags().StringVarP(&dbTables, "db-table", "t", "", "table name, multiple names separated by commas")
	_ = cmd.MarkFlagRequired("db-table")
	cmd.Flags().BoolVarP(&sqlArgs.IsEmbed, "embed", "e", true, "whether to embed gorm.model struct")
	cmd.Flags().IntVarP(&sqlArgs.JSONNamedType, "json-name-type", "j", 1, "json tags name type, 0:snake case, 1:camel case")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "output directory, default is ./dao_<time>, "+
		"if you specify the directory where the web or microservice generated by sponge, the module-name flag can be ignored")
	cmd.Flags().BoolVarP(&isIncludeInitDB, "include-init-db", "i", false, "if true, includes mysql and redis initialization code")

	return cmd
}

func runGenDaoCommand(moduleName string, isIncludeInitDB bool, codes map[string]string, outPath string) (string, error) {
	subTplName := "dao"
	r := Replacers[TplNameSponge]
	if r == nil {
		return "", errors.New("r is nil")
	}

	// setting up template information
	subDirs := []string{ // only the specified subdirectory is processed, if empty or no subdirectory is specified, it means all files
		"internal/model", "internal/cache", "internal/dao",
	}
	ignoreDirs := []string{} // specify the directory in the subdirectory where processing is ignored
	ignoreFiles := []string{ // specify the files in the subdirectory to be ignored for processing
		"init.go", "init_test.go", // internal/model
		"doc.go", "cacheNameExample.go", "cacheNameExample_test.go", // internal/cache
	}
	if isIncludeInitDB {
		ignoreFiles = []string{
			"doc.go", "cacheNameExample.go", "cacheNameExample_test.go", // internal/cache
		}
	}

	r.SetSubDirsAndFiles(subDirs)
	r.SetIgnoreSubDirs(ignoreDirs...)
	r.SetIgnoreSubFiles(ignoreFiles...)
	fields := addDAOFields(moduleName, r, codes)
	r.SetReplacementFields(fields)
	_ = r.SetOutputDir(outPath, subTplName)
	if err := r.SaveFiles(); err != nil {
		return "", err
	}

	return r.GetOutputDir(), nil
}

// set fields
func addDAOFields(moduleName string, r replacer.Replacer, codes map[string]string) []replacer.Field {
	var fields []replacer.Field

	fields = append(fields, deleteFieldsMark(r, modelFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoFile, startMark, endMark)...)
	fields = append(fields, deleteFieldsMark(r, daoTestFile, startMark, endMark)...)
	fields = append(fields, []replacer.Field{
		{
			Old: modelFileMark,
			New: codes[parser.CodeTypeModel],
		},
		{
			Old: daoFileMark,
			New: codes[parser.CodeTypeDAO],
		},
		{
			Old: selfPackageName + "/" + r.GetSourcePath(),
			New: moduleName,
		},
		{
			Old: "github.com/zhufuyi/sponge",
			New: moduleName,
		},
		{
			Old: moduleName + "/pkg",
			New: "github.com/zhufuyi/sponge/pkg",
		},
		{
			Old:             "UserExample",
			New:             codes[parser.TableName],
			IsCaseSensitive: true,
		},
	}...)

	return fields
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"

	go_ora "github.com/sijms/go-ora/v2"

	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

var (
	dsn    string
	dbType string
	query  string

	queryFrom string
	inputFrom string

	outputType string
	humanOut   bool
	jsonOut    bool
	csvOut     bool
	kvOut      bool
	xlsOut     bool
	ymlOut     bool

	output     string
	outputFile io.Writer

	debug bool

	DBConStr  string
	colCount  int
	colLength int

	sheetName string
)

// checkErrExit: Print string to stderr and exit if err is not nil
func checkErrExit(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, msg, err)
		os.Exit(1)
	}
}

// parameterErrExit: Print string to stderr and exit
func parameterErrExit(msg string) {
	_, _ = fmt.Fprintln(os.Stderr, "\x1b[1m"+msg+"\x1b[0m\n")
	usage()
	os.Exit(1)
}

// getParams: Parse flags from command line
func getParams() {
	// Rework:
	// -db { oracle }
	// -dsn
	// -query
	// -i: { sql json dir yml }
	// -if: { *.sql *.json dir }
	// -o: out, json, yml, kv, csv, xls
	// -of output { .out .json .kv .csv .xls .yml}
	// -debug

	flag.StringVar(&dbType, "db", "oracle", "Database type")
	flag.StringVar(&dsn, "dsn", "", "user:pass@dsn/service_name\nEnv: ORASQL_DSN, ORASQL_USER, ORASQL_PWD")
	flag.StringVar(&query, "query", "", "select 'column' as column_name from dual")

	flag.StringVar(&queryFrom, "i", "", "Input query from { pipe, sql, json, yml or dir }")
	flag.StringVar(&inputFrom, "if", "/dev/stdin", "Input file or directory name")

	flag.StringVar(&outputType, "o", "out", "Output { out (default), kv, json, yml, csv, xls }")
	flag.StringVar(&output, "of", "/dev/stdout", "Output file (default /dev/stdout)")

	flag.BoolVar(&debug, "debug", false, "Show column type (Work only with out type)")

	flag.Parse()

	if a := len(os.Args); a < 2 {
		parameterErrExit("Usage:")
	}

	// Syntax check -o
	switch outputType {
	case "kv":
		kvOut = true
	case "json":
		jsonOut = true
	case "csv":
		csvOut = true
	case "xls":
		xlsOut = true
	case "out":
		humanOut = true
	case "yml":
		ymlOut = true
	default:
		humanOut = true
	}

	if (jsonOut || csvOut || kvOut || xlsOut || ymlOut) && debug {
		parameterErrExit("\nInvalid -debug not allowed")
	}

	// Syntax check -i
	if queryFrom != "" {
		switch queryFrom {
		case "sql":
			if _, err := os.Stat(inputFrom); err != nil {
				parameterErrExit("\nPlease provide a valid input")
			} else {
				var tmp []byte
				tmp, err = os.ReadFile(inputFrom)
				checkErrExit("File read error: ", err)
				query = string(tmp)
			}
		case "json":
			var payloadJson struct {
				DBType   string `json:"db"`
				Dsn      string `json:"dsn"`
				User     string `json:"user"`
				Password string `json:"pwd"`
				Query    string `json:"query"`
			}
			if _, err := os.Stat(inputFrom); err != nil {
				parameterErrExit("\nPlease provide a valid inputFrom filename")
			} else {
				tmp, err := os.ReadFile(inputFrom)
				checkErrExit("Payload read error: ", err)
				err = json.Unmarshal(tmp, &payloadJson)
				checkErrExit("Json load error: ", err)
				if payloadJson.DBType == "" {
					dbType = "oracle"
				} else {
					dbType = payloadJson.DBType
				}
				dsn = fmt.Sprintf("%s:%s@%s", payloadJson.User, payloadJson.Password, payloadJson.Dsn)
				query = payloadJson.Query
			}
		case "yml":
			var payloadYAML struct {
				DBType   string `yaml:"db"`
				Dsn      string `yaml:"dsn"`
				User     string `yaml:"user"`
				Password string `yaml:"pwd"`
				Query    string `yaml:"query"`
			}
			if _, err := os.Stat(inputFrom); err != nil {
				parameterErrExit("\nPlease provide a valid inputFrom filename")
			} else {
				tmp, err := os.ReadFile(inputFrom)
				checkErrExit("Payload read error: ", err)
				err = yaml.Unmarshal(tmp, &payloadYAML)
				checkErrExit("YAML load error: ", err)
				if payloadYAML.DBType == "" {
					dbType = "oracle"
				} else {
					dbType = payloadYAML.DBType
				}
				dsn = fmt.Sprintf("%s:%s@%s", payloadYAML.User, payloadYAML.Password, payloadYAML.Dsn)
				query = payloadYAML.Query
			}
		case "dir":
		case "pipe":
			if query == "" && inputFrom == "/dev/stdin" {
				tmp, err := os.ReadFile(inputFrom)
				checkErrExit("File read error: ", err)
				query = string(tmp)
			}
		default:
			parameterErrExit("\nInvalid -i option")
		}
	}

	// Only oracle for now
	// Syntax check -db
	if dbType != "oracle" {
		dbType = "oracle"
	}

	// Syntax check -dsn
	DBConStr = os.ExpandEnv(dsn)
	if DBConStr == "" {
		oraclesqlDsn, isok := os.LookupEnv("ORASQL_DSN")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORASQL_DSN env")
		}

		oraclesqlUser, isok := os.LookupEnv("ORASQL_USER")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORASQL_USER env")
		}

		oraclesqlPwd, isok := os.LookupEnv("ORASQL_PWD")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORASQL_DSN env")
		}
		DBConStr = fmt.Sprintf("%s://%s:%s@%s", dbType, oraclesqlUser, oraclesqlPwd, oraclesqlDsn)
	} else {
		DBConStr = fmt.Sprintf("%s://%s", dbType, dsn)
	}

	if xlsOut {
		t := time.Now()
		if output == "/dev/stdout" {
			output = "Books.xlsx"
			sheetName = fmt.Sprintf("%02d%02d%04d_%02d%02d%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second())
		} else {
			pattern := `(.*):(.*)$`
			re := regexp.MustCompile(pattern)
			matches := re.FindStringSubmatch(output)
			if len(matches) > 1 {
				output = matches[1]
				sheetName = matches[2]
			} else {
				sheetName = fmt.Sprintf("%02d%02d%04d_%02d%02d%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second())
			}
		}
	}

	if output == "/dev/stdout" {
		outputFile = os.Stdout
	} else if !xlsOut {
		var err error
		outputFile, err = os.Create(output)
		checkErrExit("File creation error: ", err)
	}
}

// usage: Show usage
func usage() {
	fmt.Print(`orasql  -dsn server_url -query sql
    -db string
        oracle by default
    -dsn string
        user:pass@dsn/service_name
        Env: ORASQL_DSN, ORASQL_USER, ORASQL_PWD
    -query string
        select sysdate from dual
    -debug
        Show column type (Default output only)
    -o  { csv, json, yml, kv, xls or out (default) }
            csv   CSV Output
            json  JSON Output
            yml   YAML Output
            kv    Key/Value Output (2 columns max)
            xls   Excel file output
                  (if the xls file already exists a new sheet will be created)
    -of string
            Output file (default "/dev/stdout")
            xlsx[:sheetName] ("books.xlsx:My_Sheet" will create a Sheet named "My_Sheet" 
                              if the sheet already exists it will be deleted, so beware
                              of referenced values from this sheet to an another one)
    -i  { pipe, sql, json, yml, dir }
            pipe Read from stdin
            sql  Read the query from file
            json Read all parameters from JSON file
            yml  Read all parameters from YAML file
    -if string
        File (default "/dev/stdin")

    -i work with -if
    -o work with -of

    By default:
     -o out
     -of /dev/stdout
     -if /dev/stdin

Example:
    ./orasql  -db 'oracle' -dsn "user:pass@server/service_name" -query "select sysdate from dual"
    ./orasql  -dsn "user:pass@server/service_name" -i sql -if query.sql
    ./orasql  -i json -if sql.json
    ./orasql  -i yml -if sql.yml
    ./orasql -i json -if sql.json -o xls -of Excel.xlsx
    ./orasql -i json -if sql.json -o xls -of Excel.xlsx:sheet_name
    echo 'select sysdate from dual' |  ./orasql  -i pipe -dsn "oracle://user:pass@server/service_name"

With os environment: 
    ORACLESQL_DSN=127.0.0.1:1521/DB
    ORACLESQL_USER=user
    ORACLESQL_PWD=password
    orasql  -query "select sysdate from dual"

default output:
    SYSDATE    : 2022-01-06 18:26:37 +0000 UTC

-debug:
    SYSDATE    [DATE]           : 2022-01-06 19:26:27 +0000 UTC

-o json:
    [
        {"SYSDATE": "2022-01-06T18:21:57Z"}
    ]

-o yml:
    oraSQL:
      Lines:
        '0':
          -  SYSDATE: 2022-01-19T19:22:11Z
      LineCount: 1

-o csv:
    "SYSDATE"
    "2022-01-06 18:28:03 +0000 UTC"

-o kv: with ("select 'Date', sysdate from dual"):
    "Date": "2022-01-06T19:21:21Z"

-i json -if sql.json:
    {
      "db": "oracle"
      "dsn": "127.0.0.1:1521/DB",
      "user": "user",
      "pwd": "password",
      "query": "select sysdate from dual"
    }

-i yml -if sql.yml:
    db: "oracle"
    dsn: "127.0.0.1:1521/DB"
    user: user
    pwd: password
    query: "select sysdate from dual"
`)
}

// outputString: Write string to outputFile global var handler and call checkErrExit.
func outputString(str string) {
	_, err := fmt.Fprint(outputFile, str)
	checkErrExit("Write error: ", err)
}

func main() {
	getParams()

	DB, err := go_ora.NewConnection(DBConStr)
	checkErrExit("Driver error: ", err)
	err = DB.Open()
	checkErrExit("Open connection error: ", err)
	defer func() {
		err = DB.Close()
		checkErrExit("Connection close error: ", err)
	}()

	stmt := go_ora.NewStmt(query, DB)
	defer func() {
		err = stmt.Close()
		checkErrExit("Statement close error: ", err)
	}()

	rows, err := stmt.Query_(nil)
	checkErrExit("Query error: ", err)
	defer func() {
		err = rows.Close()
		checkErrExit("Cursor close error: ", err)
	}()

	colCount = len(rows.Columns())
	if kvOut && (colCount > 2 || colCount < 2) {
		parameterErrExit("-kv switch must have only 2 columns")
	}

	colLength = func() int {
		x := 0
		for _, v := range rows.Columns() {
			y := len(v)
			if y > x {
				x = y
			}
		}
		return x + 4
	}()

	if humanOut {
		humanoid(rows)
	} else if csvOut {
		oldFashion(rows)
	} else if jsonOut {
		robot(rows)
	} else if ymlOut {
		geek(rows)
	} else if xlsOut {
		excel(rows)
	} else {
		lazyKV(rows)
	}
}

func humanoid(dataset *go_ora.DataSet) {
	var tmp string
	baseFormat := "%-" + strconv.Itoa(colLength) + "s"
	for dataset.Next_() {
		for r, v := range dataset.CurrentRow {
			if debug {
				switch oracleType := dataset.Cols[r].DataType; oracleType {
				case 2:
					_format := baseFormat + "%-16s %s %v\n"
					if dataset.Cols[r].Precision == 38 && dataset.Cols[r].Scale == 255 {
						tmp = fmt.Sprintf(_format, dataset.Columns()[r], "["+dataset.Cols[r].DataType.String()+"]", ":", v)
						outputString(tmp)
					} else if dataset.Cols[r].Scale == 0 {
						tmp = fmt.Sprintf(_format, dataset.Columns()[r], "["+dataset.Cols[r].DataType.String()+"("+strconv.Itoa(int(dataset.Cols[r].Precision))+")"+"]", ":", v)
						outputString(tmp)
					} else {
						tmp = fmt.Sprintf(_format, dataset.Columns()[r], "["+dataset.Cols[r].DataType.String()+"("+strconv.Itoa(int(dataset.Cols[r].Precision))+","+strconv.Itoa(int(dataset.Cols[r].Scale))+")"+"]", ":", v)
						outputString(tmp)
					}
				case 1, 9, 96, 97:
					_format := baseFormat + "%-16s %s %v\n"
					_oracleTypeDecoded := dataset.Cols[r].DataType.String()

					if oracleType == 96 && dataset.Cols[r].CharsetForm == 1 {
						_oracleTypeDecoded = "CHAR"
					} else if oracleType == 96 && dataset.Cols[r].CharsetForm == 2 {
						_oracleTypeDecoded = "NCHAR"
					} else if oracleType == 1 && dataset.Cols[r].CharsetForm == 1 {
						_oracleTypeDecoded = "VARCHAR2"
					} else if oracleType == 1 && dataset.Cols[r].CharsetForm == 2 {
						_oracleTypeDecoded = "NVARCHAR2"
					}

					tmp = fmt.Sprintf(_format, dataset.Columns()[r], "["+_oracleTypeDecoded+"("+strconv.Itoa(dataset.Cols[r].MaxCharLen)+")"+"]", ":", v)
					outputString(tmp)
				default:
					_format := baseFormat + "%-16s %s %v\n"
					tmp = fmt.Sprintf(_format, dataset.Columns()[r], "["+dataset.Cols[r].DataType.String()+"]", ":", v)
					outputString(tmp)
				}
			} else {
				_format := baseFormat + ": %v\n"
				tmp = fmt.Sprintf(_format, dataset.Columns()[r], v)
				outputString(tmp)
			}
		}
	}
}

func robot(dataset *go_ora.DataSet) {
	var tmp string
	_len := colCount - 1
	tmp = "[\n"
	outputString(tmp)

	first := true
	for dataset.Next_() {
		if !first {
			tmp = "},\n  {"
			outputString(tmp)
		} else {
			first = false
			tmp = "  {"
			outputString(tmp)
		}
		for k, v := range dataset.CurrentRow {
			str, err := json.Marshal(v)
			checkErrExit("(robot) Marshall Error", err)
			if k < _len {
				tmp = fmt.Sprintf("\"%s\": %v, ", dataset.Columns()[k], string(str))
				outputString(tmp)
			} else {
				tmp = fmt.Sprintf("\"%s\": %v", dataset.Columns()[k], string(str))
				outputString(tmp)
			}
		}
	}
	if first {
		tmp = "]\n"
	} else {
		tmp = "}\n]\n"
	}
	outputString(tmp)
}

func geek(dataset *go_ora.DataSet) {
	var tmp string
	tmp = "oraSQL:\n  Lines:\n"
	outputString(tmp)
	count := 0
	for dataset.Next_() {
		tmp = fmt.Sprintf("    '%d':\n", count)
		outputString(tmp)
		for k, v := range dataset.CurrentRow {
			str, err := yaml.Marshal(v)
			checkErrExit("(geek) Marshall Error str: ", err)
			tmp = fmt.Sprintf("      -  %s: %s", dataset.Columns()[k], string(str))
			outputString(tmp)
		}
		count += 1
	}
	tmp = fmt.Sprintf("  LineCount: %d\n", count)
	outputString(tmp)
}

func oldFashion(dataset *go_ora.DataSet) {
	var tmp string
	_len := colCount - 1
	for k, v := range dataset.Columns() {
		if k < _len {
			tmp = fmt.Sprintf(`"%s",`, v)
			outputString(tmp)
		} else {
			tmp = fmt.Sprintf("\"%s\"\n", v)
			outputString(tmp)
		}
	}
	for dataset.Next_() {
		for k, v := range dataset.CurrentRow {
			if k < _len {
				if v == nil {
					v = "NULL"
					tmp = fmt.Sprintf(`%v,`, v)
					outputString(tmp)
				} else if dataset.Cols[k].DataType == 2 || dataset.Cols[k].DataType == 4 {
					tmp = fmt.Sprintf(`%v,`, v)
					outputString(tmp)
				} else {
					tmp = fmt.Sprintf(`"%v",`, v)
					outputString(tmp)
				}
			} else {
				if v == nil {
					v = "NULL"
					tmp = fmt.Sprintf("%v\n", v)
					outputString(tmp)
				} else if dataset.Cols[k].DataType == 2 || dataset.Cols[k].DataType == 4 {
					tmp = fmt.Sprintf("%v\n", v)
					outputString(tmp)
				} else {
					tmp = fmt.Sprintf("\"%v\"\n", v)
					outputString(tmp)
				}
			}
		}
	}
}

func lazyKV(dataset *go_ora.DataSet) {
	var tmp string
	for dataset.Next_() {
		str0, err := json.Marshal(dataset.CurrentRow[0])
		checkErrExit("(kv) Marshall Error", err)
		str1, err := json.Marshal(dataset.CurrentRow[1])
		checkErrExit("(kv) Marshall Error", err)
		tmp = fmt.Sprintf("%s: %s\n", str0, str1)
		outputString(tmp)
	}
}

func excel(dataset *go_ora.DataSet) {
	isCreated := false
	var f *excelize.File
	if xlsOut {
		if _, err := os.Stat(output); err != nil {
			f = excelize.NewFile()
			isCreated = true
		} else {
			f, err = excelize.OpenFile(output)
			checkErrExit("Cannot open/read or not type of xlsx file", err)
		}
	}

	sheetIndex, _ := f.GetSheetIndex(sheetName)
	if sheetIndex != -1 {
		f.DeleteSheet(sheetName)
	}
	sheetIndex, _ = f.NewSheet(sheetName)
	f.SetActiveSheet(sheetIndex)

	style, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Color: "#FF0000"}})
	checkErrExit("(excel) Create style error", err)

	for k, v := range dataset.Columns() {
		int2ColRow, err := excelize.CoordinatesToCellName(k+1, 1)
		checkErrExit("(excel) int2ColRow", err)
		err = f.SetCellValue(sheetName, int2ColRow, v)
		checkErrExit("(excel)[1] Set cell value error", err)
		err = f.SetCellStyle(sheetName, int2ColRow, int2ColRow, style)
		checkErrExit("(excel) Set style error", err)
	}

	row := 1
	for dataset.Next_() {
		row += 1
		for k, v := range dataset.CurrentRow {
			int2ColRow, err := excelize.CoordinatesToCellName(k+1, row)
			checkErrExit("(excel) int2ColRow", err)
			err = f.SetCellValue(sheetName, int2ColRow, v)
			checkErrExit("(excel)[2] Set cell value error", err)
		}
	}

	if isCreated {
		err = f.SaveAs(output)
		checkErrExit("(excel)[SaveAs] Write error", err)
	} else {
		err = f.Save()
		checkErrExit("(excel)[Save] Write error", err)
	}
}

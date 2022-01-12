package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sijms/go-ora/v2"
	"github.com/xuri/excelize/v2"
	"io"
	"os"
	"strconv"
)

var (
	humanOut bool
	jsonOut  bool
	csvOut   bool
	kvOut    bool
	excelOut bool
	debug    bool

	DBConStr  string
	query     string
	colCount  int
	colLength int

	outputFile io.Writer
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
	_, _ = fmt.Fprintln(os.Stderr, msg)
	usage()
	os.Exit(1)
}

// getParams: Parse flags from command line
func getParams() {
	var (
		dsn           string
		queryFromFile string
		output        string
		payload       string
	)

	// Rework:
	// -of output { .out .json .kv .csv .xls }
	// -if input  { .sql .json .ini }
	// -ot outputType: human, json, kv, csv, xls
	// -query
	// -dsn
	// -debug

	flag.StringVar(&dsn, "dsn", "", "oracle://user:pass@dsn/service_name\nEnv: ORACLESQL_DSN, ORACLESQL_USER, ORACLESQL_PWD")
	flag.StringVar(&query, "query", "", "select 'column' as column_name from dual")
	flag.StringVar(&queryFromFile, "file", "/dev/stdin", "Input query from file")
	flag.StringVar(&payload, "payload", "", "Input payload Json from file")
	flag.StringVar(&output, "output", "/dev/stdout", "Output file")
	flag.BoolVar(&jsonOut, "json", false, "JSON Output")
	flag.BoolVar(&csvOut, "csv", false, "CSV Output")
	flag.BoolVar(&kvOut, "kv", false, "Key/Value Output (2 columns max)")
	flag.BoolVar(&excelOut, "excel", false, "Excel Output")
	flag.BoolVar(&debug, "debug", false, "Show column type (Default output only)")
	flag.Parse()

	// Payload must be the first
	// Cause it define some mandatory vars
	if payload != "" {
		var payloadJson struct {
			Dsn      string `json:"dsn"`
			User     string `json:"user"`
			Password string `json:"password"`
			Query    string `json:"query"`
		}
		if _, err := os.Stat(payload); err != nil {
			parameterErrExit("\nPlease provide a valid payload filename")
		} else {
			tmp, err := os.ReadFile(payload)
			checkErrExit("Payload read error: ", err)
			err = json.Unmarshal(tmp, &payloadJson)
			checkErrExit("Json load error: ", err)
			dsn = fmt.Sprintf("oracle://%s:%s@%s", payloadJson.User, payloadJson.Password, payloadJson.Dsn)
			query = payloadJson.Query
		}
	}

	DBConStr = os.ExpandEnv(dsn)
	if DBConStr == "" {
		oraclesqlDsn, isok := os.LookupEnv("ORACLESQL_DSN")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORACLESQL_DSN env")
		}

		oraclesqlUser, isok := os.LookupEnv("ORACLESQL_USER")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORACLESQL_USER env")
		}

		oraclesqlPwd, isok := os.LookupEnv("ORACLESQL_PWD")
		if !isok {
			parameterErrExit("\nMissing -dsn option or ORACLESQL_DSN env")
		}
		DBConStr = fmt.Sprintf("oracle://%s:%s@%s", oraclesqlUser, oraclesqlPwd, oraclesqlDsn)
	}

	if jsonOut && csvOut {
		parameterErrExit("\nPlease select only one output -json or -csv")
	}

	if kvOut && csvOut {
		parameterErrExit("\n-kv not allowed with -csv")
	}
	if !jsonOut && !csvOut && !kvOut && !excelOut || debug {
		humanOut = true
	}

	if (jsonOut || csvOut || kvOut || excelOut) && debug {
		parameterErrExit("\n-debug not allowed with other output")
	}

	if output == "/dev/stdout" {
		outputFile = os.Stdout
	} else {
		var err error
		outputFile, err = os.Create(output)
		checkErrExit("File creation error: ", err)
	}

	if query == "" {
		if _, err := os.Stat(queryFromFile); err != nil {
			parameterErrExit("\nPlease provide a valid filename")
		} else {
			var tmp []byte
			tmp, err = os.ReadFile(queryFromFile)
			checkErrExit("File read error: ", err)
			query = string(tmp)
		}
	}
}

// usage: Show usage
func usage() {
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println(os.Args[0], ` -dsn server_url -query sql`)
	flag.PrintDefaults()
	fmt.Println()
	fmt.Print("Example:\n\n")
	fmt.Println("  ", os.Args[0], ` -dsn "oracle://user:pass@server/service_name" -query "select sysdate from dual"`)
	fmt.Println("  ", os.Args[0], ` -dsn "oracle://user:pass@server/service_name" -file query.sql`)
	fmt.Println("  ", os.Args[0], ` -payload payload.json`)
	fmt.Println("   echo 'select sysdate from dual' | ", os.Args[0], ` -dsn "oracle://user:pass@server/service_name"`)
	fmt.Println("\nWith os.env: ")
	fmt.Println(`   ORACLESQL_DSN=127.0.0.1:1521/DB ORACLESQL_USER=user ORACLESQL_PWD=password `, os.Args[0], ` -query "select sysdate from dual"`)
	fmt.Print("\ndefault output:\n", `  SYSDATE    : 2022-01-06 18:26:37 +0000 UTC`, "\n")
	fmt.Print("\n-debug:\n", `  SYSDATE    [DATE]           : 2022-01-06 19:26:27 +0000 UTC`, "\n")
	fmt.Print("\n-json:\n", `  [
    {"SYSDATE": "2022-01-06T18:21:57Z"}
  ]`, "\n")
	fmt.Print("\n-csv:\n", `  "SYSDATE"
  "2022-01-06 18:28:03 +0000 UTC"`, "\n")
	fmt.Print("\n-kv with (\"select 'Date', sysdate from dual\"):\n", `  "Date": "2022-01-06T19:21:21Z"`, "\n")
	fmt.Print("\n-payload:", `
With json file:
  {
    "dsn": "127.0.0.1:1521/DB",
    "user": "user",
    "password": "password",
    "query": "select sysdate from dual"
  }`, "\n")
}

// outputString: Write string to outputFile global var handler and call checkErrExit.
func outputString(str string) {
	_, err := fmt.Fprintf(outputFile, str)
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
	} else if excelOut {
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
				// go-ora OracleType
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
	tmp = fmt.Sprint("[\n")
	outputString(tmp)

	first := true
	for dataset.Next_() {
		if !first {
			tmp = fmt.Sprint("},\n  {")
			outputString(tmp)
		} else {
			first = false
			tmp = fmt.Sprint("  {")
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
	tmp = fmt.Sprint("}\n]\n")
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

// Pour le fun...
func excel(dataset *go_ora.DataSet) {
	f := excelize.NewFile()
	// TODO: Manage output file, if the file already exists add a sheet instead of creating a new excel file.
	index := f.NewSheet("Sheet1")
	style, err := f.NewStyle(`{"font":{"bold":true,"color":"#FF0000"}}`)
	checkErrExit("(excel) Create style error", err)

	for k, v := range dataset.Columns() {
		int2ColRow, err := excelize.CoordinatesToCellName(k+1, 1)
		checkErrExit("(excel) int2ColRow", err)
		err = f.SetCellValue("Sheet1", int2ColRow, v)
		checkErrExit("(excel)[1] Set cell value error", err)
		err = f.SetCellStyle("Sheet1", int2ColRow, int2ColRow, style)
		checkErrExit("(excel) Set style error", err)
	}

	row := 1
	for dataset.Next_() {
		row += 1
		for k, v := range dataset.CurrentRow {
			int2ColRow, err := excelize.CoordinatesToCellName(k+1, row)
			checkErrExit("(excel) int2ColRow", err)
			err = f.SetCellValue("Sheet1", int2ColRow, v)
			checkErrExit("(excel)[1] Set cell value error", err)
		}
	}
	f.SetActiveSheet(index)
	err = f.SaveAs("Book1.xlsx")
	checkErrExit("(excel) Write error", err)
}

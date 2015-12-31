package nsqscript

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

/* BNF

<prog>      ::= (<statement>)*
<statement> ::= <instr> <space> (<expr>)*
<instr>     ::= "pause" | "unpause" | "ping" | "empty" | "delete"
<expr>      ::= <ident> <space> <string>
<ident>     ::= "ip" | "channel" | "topic"
<space>     ::= "\s"
<string>    ::= (<char>)+
<char>      ::= [a-zA-Z0-9\.]

*/

type Token struct {
	Type TokenType
	Text string
	Num  int
}

type TokenType int

type Expr struct {
	Name  string
	Value string
}

const (
	tokenError TokenType = iota
	tokenEOF
	tokenEOL

	// Commands
	tokenChannel
	tokenTopic
	tokenIP
	tokenPause
	tokenUnpause
	tokenPing
	tokenEmpty
	tokenDelete

	// Primary
	tokenLabel
	tokenValue
)

const (
	tstrEOL     = "\n"
	tstrChannel = "channel"
	tstrTopic   = "topic"
	tstrIP      = "ip"
	tstrPause   = "pause"
	tstrUnpause = "unpause"
	tstrPing    = "ping"
	tstrEmpty   = "empty"
	tstrDelete  = "delete"
)

var (
	currToken = Token{0, "", 0}
	//tokenList   = []Token{}
	tokenList   = []string{}
	symbolTable = map[string]string{}
)

func nextToken() {
	tlen := len(tokenList)
	if tlen > 0 {
		s := tokenList[0:1][0]
		tokenList = tokenList[1:]
		switch s {
		case tstrEOL:
			currToken = Token{tokenEOL, "", 0}
		case tstrIP:
			currToken = Token{tokenIP, "", 0}
		case tstrTopic:
			currToken = Token{tokenTopic, "", 0}
		case tstrChannel:
			currToken = Token{tokenChannel, "", 0}
		case tstrPing:
			currToken = Token{tokenPing, "", 0}
		case tstrPause:
			currToken = Token{tokenPause, "", 0}
		default:
			//if isIPAddress(s) {
			//	symbolTable[s] = ""
			//	currToken = Token{tokenLabel, "ip", s}
			//} else {
			//symbolTable[s] = ""
			currToken = Token{tokenValue, s, 0}

			//fmt.Println("Syntax error in:", s)
			//}
		}
	} else {
		currToken = Token{tokenEOF, "", 0}
	}
}

func consume(expected TokenType) {
	if currToken.Type == expected {
		nextToken()
	} else {
		fmt.Println("Expected", expected, "not found.")
	}
}

func ParseLine(line string) string {
	if line[len(line)-1:] != "\n" {
		line += "\n"
	}
	t := []byte{}
	bLine := []byte(line)
	for _, b := range bLine {
		if b == 32 || b == 9 {
			tokenList = append(tokenList, string(t))
			t = []byte{}
		} else if b == 10 {
			if len(t) > 0 {
				tokenList = append(tokenList, string(t))
			}
			t = []byte{10}
			tokenList = append(tokenList, string(t))
			t = []byte{}
		} else {
			t = append(t, b)
		}
	}
	nextToken()
	stmts := buildStatements()
	resultChan := make(chan string)
	go execStatementList(stmts, resultChan)
	result := <-resultChan
	close(resultChan)
	return result
}

func ParseScript(file *os.File, resultsChan chan string) {
	sc := bufio.NewScanner(file)
	sc.Split(bufio.ScanBytes)
	t := []byte{}
	for sc.Scan() {
		b := sc.Bytes()[0]
		if b == 32 || b == 9 {
			tokenList = append(tokenList, string(t))
			t = []byte{}
		} else if b == 10 {
			if len(t) > 0 {
				tokenList = append(tokenList, string(t))
			}
			t = []byte{10}
			tokenList = append(tokenList, string(t))
			t = []byte{}
		} else {
			t = append(t, b)
		}
	}
	nextToken()
	stmts := buildStatements()
	execStatementList(stmts, resultsChan)
	close(resultsChan)
}

func buildStatements() [][]Expr {
	stmts := [][]Expr{}
	newStmt := []Expr{}
	newExpr := Expr{}

Loop:
	for {
		switch currToken.Type {
		case tokenEOF:
			break Loop
		case tokenEOL:
			if len(newStmt) > 0 {
				stmts = append(stmts, newStmt)
				newStmt = []Expr{}
			}
			nextToken()
		case tokenValue:
			newExpr.Value = currToken.Text
			newStmt = append(newStmt, newExpr)
			nextToken()
		case tokenIP:
			newExpr = Expr{Name: "ip"}
			nextToken()
		case tokenTopic:
			newExpr = Expr{Name: "topic"}
			nextToken()
		case tokenChannel:
			newExpr = Expr{Name: "channel"}
			nextToken()
		case tokenPing:
			newExpr = Expr{Name: "ping"}
			newStmt = []Expr{newExpr}
			nextToken()
		case tokenPause:
			newExpr = Expr{Name: "pause"}
			newStmt = []Expr{newExpr}
			nextToken()
		}
	}
	return stmts
}

func execStatementList(stmtList [][]Expr, resultsChan chan string) {
	for _, j := range stmtList {
		result := execStatement(j)
		resultsChan <- result
	}
}

func execStatement(stmt []Expr) string {
	uri := ""
	switch stmt[0].Name {
	case "ping":
		uri = stmt[1].Value + "/ping"
	case "pause":
		if len(stmt) == 3 { // "topic"
			uri = stmt[1].Value + "/pause/topic?topic=" + stmt[2].Value
		} else { // "channel"
			channel := findExpr(stmt, "channel")
			topic := findExpr(stmt, "topic")
			uri = stmt[1].Value + "/pause/channel?topic=" + topic.Value + "&channel=" + channel.Value
		}
	}
	uri = "http://" + uri
	return request(uri)
}

func request(u string) string {
	req, err := http.NewRequest("POST", u, nil)
	if err != nil {
		return fmt.Sprint("Error building request:", err)
	}
	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Sprint("Error sending request:", err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprint("Error reading body:", err)
	}

	return string(contents)
}

func isIPAddress(s string) bool {
	ip := net.ParseIP(s)
	if ip != nil {
		return true
	}
	return false
}

func findExpr(statement []Expr, name string) Expr {
	for _, j := range statement {
		if j.Name == name {
			return j
		}
	}
	return Expr{}
}

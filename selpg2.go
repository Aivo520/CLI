package main

import (
	"fmt"
	"os"
	flag  "github.com/spf13/pflag"
	"os/exec"
	"io"
	// "strconv"
	"bufio"
)

type selpg_args struct {
	start_page int
	end_page int
	in_filename string
	page_len int
	page_type string
	print_dest string
}
var progname string

func main(){
	// ac := len(os.Args)
	progname = os.Args[0]

	var sa selpg_args
	flag.Usage = func() {
		fmt.Printf("\nUSAGE: %s -sstart_page -eend_page [ -f | -llines_per_page ] [ -ddest ] [ in filename ]\n", progname)
		flag.PrintDefaults()
	}
	flag.IntVarP(&sa.start_page, "start_page", "s", -1, "defaults to -1")
	flag.IntVarP(&sa.end_page, "end_page", "e", -1, "defaults to -1")
	flag.IntVarP(&sa.page_len, "page_len", "l", 72, "defaults to 72")
	flag.StringVarP(&sa.page_type, "page_type", "f", "l", "defaults to l")
	flag.Lookup("page_type").NoOptDefVal = "f"
	flag.StringVarP(&sa.print_dest, "print_dest", "d", "", "defaults to nothing")
	flag.Parse()

	if flag.NArg() == 1 {
		sa.in_filename = flag.Args()[0]
		_, err := os.Stat(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not open input file \"%s\"\n", progname, sa.in_filename)
			os.Exit(1)
		}
	}

	process_input(&sa)
}


func process_input(sa *selpg_args) {
	var fin *os.File
	var err error
	page_ctr := 1
	
	// input source
	if len(sa.in_filename) == 0 {
		fin = os.Stdin
	} else {
		fin, err = os.Open(sa.in_filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not open input file \"%s\"\n", progname, sa.in_filename)
			panic(err)
			os.Exit(12)
		}
		defer fin.Close()
	}
	buf := bufio.NewReader(fin)

	// output dest
	var fout io.WriteCloser
	if len(sa.print_dest) == 0 {
		fout = os.Stdout
	} else {
		cmd := exec.Command("lp", "-d", sa.print_dest)
		cmd.Stdout, err = os.OpenFile(sa.print_dest, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		fout, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not pipe to \"%s\"\n", progname, sa.print_dest)
			panic(err)
			os.Exit(13)
		}
		cmd.Start()
	}

// loop to read data
	if sa.page_type == "l" {
		line_ctr := 0
		// page_ctr := 1
		for {
			line_ctr += 1
			if line_ctr > sa.page_len {
				page_ctr += 1
				line_ctr = 1
			}
			if (page_ctr < sa.start_page) || (page_ctr > sa.end_page) {
				break
			}
			line, reading_error := buf.ReadString('\n')
			if reading_error != nil {
				break
			}
			_, writing_error := fout.Write([]byte(line))
			if writing_error != nil {
				panic(writing_error)
			}
		}
	} else {
		// page_ctr := 1
		var bs []byte
		var b byte
		for {
			b, err = buf.ReadByte()
			if err == io.EOF {
				break
			}
			if b == '\f' {
				page_ctr += 1
			}
			bs = append(bs, b)
			if page_ctr >= sa.start_page && page_ctr <= sa.end_page {
				_, writing_error := fout.Write([]byte(bs))
				if writing_error != nil {
					panic(writing_error)
				}
			} else {
				break
			}
		}
	}
	// check
	if page_ctr < sa.start_page || page_ctr < sa.end_page {
		fmt.Fprintf(os.Stderr, "page number appear to error!\n")
	}
}

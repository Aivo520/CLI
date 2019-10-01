package main

import (
	"fmt"
	"os"
	// "flag"
	"os/exec"
	"io"
	"strconv"
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
	ac := len(os.Args)
	progname = os.Args[0]

	var sa selpg_args
	sa.start_page = -1
	sa.end_page = -1
	sa.page_len = 72
	sa.page_type = "l"
	
	// fmt.Printf("n_args: %d\n", ac)
	// fmt.Println(os.Args)

	process_args(ac, os.Args, &sa)
	process_input(&sa)
}

func process_args(ac int, av []string, sa *selpg_args) {
	var s1, s2 string

	if ac < 3 {
		fmt.Fprintf(os.Stderr, "%s: not enough arguments\n", progname)
		usage()
		os.Exit(1)
	}

	// get the start page
	s1 = av[1]
	if s1[:2] != "-s" {
		fmt.Fprintf(os.Stderr, "%s: 1st arg should be -sstart_page\n", progname)
		usage()
		os.Exit(2)
	}
	i, err := strconv.Atoi(s1[2:])
	if err != nil || i < 1 {
		fmt.Fprintf(os.Stderr, "%s: invalid start page %s\n", progname, s1[2:])
		usage()
		os.Exit(3)
	}
	sa.start_page = i
	
	// get the end page
	s1 = av[2]
	if s1[:2] != "-e" {
		fmt.Fprintf(os.Stderr, "%s: 2nd arg should be -eend_page\n", progname)
		usage()
		os.Exit(4)
	}
	i, err = strconv.Atoi(s1[2:])
	if err != nil || i < 1 {
		fmt.Fprintf(os.Stderr, "%s: invalid end page %s\n", progname, s1[2:])
		usage()
		os.Exit(5)
	}
	sa.end_page = i

	// optional parameters
	argno := 3
	for ;argno <= (ac - 1) && av[argno][0] == '-'; {
		s1 = av[argno]
		switch s1[1] {
			// page length
			case 'l':
				s2 = s1[2:]
				i, err = strconv.Atoi(s2)
				if err != nil || i < 1 {
					fmt.Fprintf(os.Stderr, "%s: invalid page length %s\n", progname, s2)
					usage()
					os.Exit(6)
				}
				sa.page_len = i
				argno += 1
			// if -f
			case 'f':
				if s1 != "-f" {
					fmt.Fprintf(os.Stderr, "%s: option should be \"-f\"\n", progname, s2)
					usage()
					os.Exit(7)
				}
				sa.page_type = "f"
				argno += 1
			// print destination
			case 'd':
				s2 = s1[2:]
				if len(s2) < 1 {
					fmt.Fprintf(os.Stderr, "%s: -d option require a printer destination\n", progname, s2)
					usage()
					os.Exit(8)
				}
				sa.print_dest = s2
				argno += 1
			// default
			default:
				fmt.Fprintf(os.Stderr, "%s: unknown option %s\n", progname, s1)
				usage()
				os.Exit(9)
		}
	}

	// if one more arg, must be input file name
	if argno <= ac - 1 {
		sa.in_filename = av[argno]
		pfile, reading_error := os.Open(sa.in_filename)
		if reading_error != nil {
			panic("input file can not be found or be read\n")
		}
		defer pfile.Close()
	}

	// final check
	if sa.start_page <= 0 || sa.end_page <= 0 || sa.end_page < sa.start_page {
		panic("panic: page is not valid\n")
	}
	if sa.page_len <= 1 || (sa.page_type != "l" && sa.page_type != "f") {
		panic("panic: page property is not valid\n")
	}
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


func usage() {
	fmt.Printf("\nUSAGE: %s -sstart_page -eend_page [ -f | -llines_per_page ] [ -ddest ] [ in filename ]\n", progname)
}

实验内容：
使用 golang 开发 Linux 命令行实用程序 中的 selpg


源代码：
selpg.go, selpg2.go


运行：
$ go build selpg.go
$ ./selpg ...
或者：
$ go build selpg2.go
$ ./selpg2 ...


设计模块：
1. 定义参数结构体，包括start_page, end_page, page_len, page_type, print_dest，代码如下：
'''
type selpg_args struct {
	start_page int
	end_page int
	in_filename string
	page_len int
	page_type string
	print_dest string
}
'''
2. 获取输入参数，selpg.go使用传统的os.Args以及字符串处理方式获取，而selpg2.go使用pflag包获取
  传统获取方式：
    将selpg.c的代码用go语言语法代替，通过切片的方式获取数字部分，然后使用strconv.Atoi获取数值
    '''
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
  '''
    
  pflag包获取方式：
    借鉴网上的代码，使用IntVarP和StringVarP获取，-l与-f部分使用Lookup获取sa.page_type地址以进行赋值
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
  // 如果有输入文件...
	if flag.NArg() == 1 {
		sa.in_filename = flag.Args()[0]
		_, err := os.Stat(flag.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not open input file \"%s\"\n", progname, sa.in_filename)
			os.Exit(1)
		}
	}
  
3. 运行过程：
  1) 获取in流（可能来自终端os.Stdin，也可能来自文件）：
  if len(sa.in_filename) == 0 {
		fin = os.Stdin
	} else {
    // 使用os.Open打开文件，这种打开方式意味着只读
		fin, err = os.Open(sa.in_filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not open input file \"%s\"\n", progname, sa.in_filename)
			panic(err)
			os.Exit(12)
		}
		defer fin.Close()
	}
  // 创建读的缓冲区
	buf := bufio.NewReader(fin)
  
  2) 获取输出流（可能是终端，也可能是打印机【然而我并没有打印机，所以无法测试该情况】）
  // output dest
	var fout io.WriteCloser
	if len(sa.print_dest) == 0 {
		fout = os.Stdout
	} else {
    // exec.Command用以调用linux的库命令
		cmd := exec.Command("lp", "-d", sa.print_dest)
    // 将输出流与printer挂钩
		cmd.Stdout, err = os.OpenFile(sa.print_dest, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		fout, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cound not pipe to \"%s\"\n", progname, sa.print_dest)
			panic(err)
			os.Exit(13)
		}
    // 以非阻塞方式执行
		cmd.Start()
	}
  3) 打印
  // loop to read data
	if sa.page_type == "l" {
    // -l模式，按行读取
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
    // -f模式，按照换页符读取
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
  

测试结果：
执行"使用selpg"部分的用例，涉及到打印机部分无法测试。其它部分基本符合要求。
但是有一点，在我用vi 新建一个data.txt文件时，使用-f模式输出，会出现重复输出好几次。我测试了一下selpg.c，发现结果一样。
所以有可能是文档的问题，因为我测试了一个几十页的.doc文档，发现只有3个换页符。



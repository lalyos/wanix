package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"

	"tractor.dev/toolkit-go/engine/cli"
	"tractor.dev/toolkit-go/engine/fs/fsutil"
	"tractor.dev/toolkit-go/engine/fs/watchfs"
)

func echoCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "echo [text]...",
		Run: func(ctx *cli.Context, args []string) {
			io.WriteString(ctx, strings.Join(args, " "))
		},
	}
	return cmd
}

func openCmd() *cli.Command {
	var openWatch *watchfs.Watch
	cmd := &cli.Command{
		Usage: "open <appname>",
		Args:  cli.ExactArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			var path string
			if exists, _ := fsutil.Exists(os.DirFS("/"), fmt.Sprintf("sys/app/%s", args[0])); exists {
				path = fmt.Sprintf("sys/app/%s", args[0])
			} else if exists, _ := fsutil.Exists(os.DirFS("/"), fmt.Sprintf("app/%s", args[0])); exists {
				path = fmt.Sprintf("app/%s", args[0])
			} else {
				fmt.Fprintln(ctx, "app not found")
				return
			}
			if openWatch != nil {
				openWatch.Close()
			}

			// todo: port from afero to engine/fs so watchfs works
			// --
			// var err error
			// var firstWrite bool
			// if args[0] == "jazz-todo" {
			// 	openWatch, err = watchfs.WatchFile(fs, "app/jazz-todo/view.jsx", &watchfs.Config{
			// 		Handler: func(e watchfs.Event) {
			// 			if e.Type == watchfs.EventWrite && len(e.Path) > len(path) {
			// 				if !firstWrite {
			// 					firstWrite = true
			// 					return
			// 				}
			// 				js.Global().Get("wanix").Get("loadApp").Invoke("main")
			// 			}
			// 		},
			// 	})
			// } else {
			// 	openWatch, err = watchfs.WatchFile(fs, path, &watchfs.Config{
			// 		Recursive: true,
			// 		Handler: func(e watchfs.Event) {
			// 			if e.Type == watchfs.EventWrite && len(e.Path) > len(path) {
			// 				if !firstWrite {
			// 					firstWrite = true
			// 					return
			// 				}
			// 				js.Global().Get("wanix").Get("loadApp").Invoke("main")
			// 			}
			// 		},
			// 	})
			// }
			// if err != nil {
			// 	fmt.Fprintf(t, "%s\n", err)
			// 	return
			// }
			js.Global().Get("wanix").Get("loadApp").Invoke("main", fmt.Sprintf("/-/%s/", path), true)
		},
	}
	return cmd
}

func mtimeCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "mtime <path>",
		Args:  cli.ExactArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			fi, err := os.Stat(args[0])
			if err != nil {
				fmt.Fprintf(ctx, "%s\n", err)
				return
			}
			fmt.Fprintf(ctx, "%s\n", fi.ModTime())
		},
	}
	return cmd
}

func lsCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "ls [path]",
		Args:  cli.MaxArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			var path string
			if len(args) > 0 {
				path = absPath(args[0])
			} else {
				path, _ = os.Getwd()
			}

			fi, err := os.ReadDir(path)
			if checkErr(ctx, err) {
				return
			}
			for _, entry := range fi {
				dirSuffix := ' '
				if entry.IsDir() {
					dirSuffix = '/'
				}
				info, _ := entry.Info()
				fmt.Fprintf(ctx, "%v %-4d %s%c\n", info.Mode(), info.Size(), info.Name(), dirSuffix)
			}
		},
	}
	return cmd
}

func cdCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "cd <path>",
		Args:  cli.ExactArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			exists, err := fsutil.DirExists(os.DirFS("/"), unixToFsPath(args[0]))
			if checkErr(ctx, err) {
				return
			}

			if !exists {
				fmt.Fprintln(ctx, "invalid directory")
				return
			}

			path := absPath(args[0])
			if path == "." {
				return
			}
			if checkErr(ctx, os.Chdir(path)) {
				return
			}

		},
	}
	return cmd
}

func catCmd() *cli.Command {
	cmd := &cli.Command{
		Usage: "cat <path>...",
		Args:  cli.MinArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			// todo: multiple files
			d, err := os.ReadFile(absPath(args[0]))
			if checkErr(ctx, err) {
				return
			}
			ctx.Write(d)
			io.WriteString(ctx, "\n")
		},
	}
	return cmd
}

func reloadCmd() *cli.Command {
	return &cli.Command{
		Usage: "reload",
		Args:  cli.ExactArgs(0),
		Run: func(ctx *cli.Context, args []string) {
			fmt.Println("TODO: Unimplemented")
			// js.Global().Get("wanix").Get("reload").Invoke()
		},
	}
}

func downloadCmd() *cli.Command {
	return &cli.Command{
		Usage: "dl <path>",
		Args:  cli.ExactArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			fmt.Println("TODO: Unimplemented")
			// js.Global().Get("wanix").Get("download").Invoke(args[0])
		},
	}
}

func touchCmd() *cli.Command {
	return &cli.Command{
		Usage: "touch <path>...",
		Args:  cli.MinArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			// TODO: multiple files, options for updating a/mtimes
			err := os.WriteFile(absPath(args[0]), []byte{}, 0644)
			if checkErr(ctx, err) {
				return
			}
		},
	}
}

func removeCmd() *cli.Command {
	var recursive bool

	cmd := &cli.Command{
		Usage: "rm [-r] <path>...",
		Args:  cli.MinArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			// TODO: multiple files
			if recursive {
				err := os.RemoveAll(absPath(args[0]))
				if checkErr(ctx, err) {
					return
				}
			} else {
				if isdir, err := fsutil.IsDir(os.DirFS("/"), unixToFsPath(args[0])); isdir {
					fmt.Fprintf(ctx, "cant remove file %s: is a directory\n(try using the `-r` flag)\n", absPath(args[0]))
					return
				} else if checkErr(ctx, err) {
					return
				}

				// TODO: fs.Remove gives the wrong error if trying to delete a readonly file,
				// (should be Operation not permitted)
				err := os.Remove(absPath(args[0]))
				if checkErr(ctx, err) {
					return
				}
			}
		},
	}

	cmd.Flags().BoolVar(&recursive, "r", false, "Remove recursively")
	return cmd
}

func mkdirCmd() *cli.Command {
	return &cli.Command{
		Usage: "mkdir <path>",
		Args:  cli.ExactArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			// TODO: support MkdirAll
			err := os.Mkdir(absPath(args[0]), 0755)
			if checkErr(ctx, err) {
				return
			}
		},
	}
}

func moveCmd() *cli.Command {
	return &cli.Command{
		Usage: "mv SOURCE DEST | SOURCE... DIRECTORY",
		Args:  cli.MinArgs(2),
		Short: "Rename SOURCE to DEST, or move multiple SOURCE(s) to DIRECTORY.",
		Run: func(ctx *cli.Context, args []string) {
			// TODO: prevent file overwrite if dest file already exits (should this already be an error?)
			// TODO: error when dest directory doesn't exist and args.len > 2
			isdir, err := fsutil.DirExists(os.DirFS("/"), unixToFsPath(args[len(args)-1]))
			if checkErr(ctx, err) {
				return
			}
			if isdir {
				// move all paths into this directory
				dir := absPath(args[len(args)-1])
				for _, path := range args[:len(args)-1] {
					src := filepath.Base(path)
					dest := filepath.Join(dir, src)
					err := os.Rename(absPath(path), absPath(dest))
					if err != nil {
						io.WriteString(ctx, fmt.Sprintln(err))
						continue
					}
				}
			} else {
				err := os.Rename(absPath(args[0]), absPath(args[1]))
				if checkErr(ctx, err) {
					return
				}
			}
		},
	}
}

func copyCmd() *cli.Command {
	var recursive bool

	cmd := &cli.Command{
		Usage: "cp [-r] <SOURCE DEST | SOURCE... DIRECTORY> ",
		Args:  cli.MinArgs(2),
		Short: "Copy SOURCE to DEST, or multiple SOURCE(s) to DIRECTORY.",
		Run: func(ctx *cli.Context, args []string) {
			// TODO: handle copying directories
			isdir, err := fsutil.DirExists(os.DirFS("/"), unixToFsPath(args[len(args)-1]))
			if checkErr(ctx, err) {
				return
			}
			if isdir {
				// copy all paths to this directory
				dir := absPath(args[len(args)-1])

				for _, path := range args[:len(args)-1] {
					srcName := filepath.Base(path)
					dest := filepath.Join(dir, srcName)

					srcIsDir, err := fsutil.IsDir(os.DirFS("/"), unixToFsPath(path))
					if checkErr(ctx, err) {
						continue
					}

					if srcIsDir {
						if !recursive {
							io.WriteString(ctx, fmt.Sprintf("-r not specified; omitting directory '%s'\n", path))
							continue
						}

						err = os.MkdirAll(absPath(dest), 0755)
						if checkErr(ctx, err) {
							continue
						}

						entries, err := os.ReadDir(absPath(path))
						if checkErr(ctx, err) {
							continue
						}

						for _, e := range entries {
							cli.Execute(ctx, copyCmd(), []string{"-r", filepath.Join(path, e.Name()), dest})
							// commands["cp"](t, fs, []string{"-r", filepath.Join(path, e.Name()), dest})
						}
					} else {
						content, err := os.ReadFile(absPath(path))
						if checkErr(ctx, err) {
							continue
						}
						err = os.WriteFile(absPath(dest), content, 0644)
						if checkErr(ctx, err) {
							continue
						}
					}
				}
			} else {
				content, err := os.ReadFile(absPath(args[0]))
				if checkErr(ctx, err) {
					return
				}

				err = os.WriteFile(absPath(args[1]), content, 0644)
				if checkErr(ctx, err) {
					return
				}
			}
		},
	}

	cmd.Flags().BoolVar(&recursive, "r", false, "Copy recursively")
	return cmd
}

func pwdCmd() *cli.Command {
	return &cli.Command{
		Usage: "pwd",
		Args:  cli.ExactArgs(0),
		Run: func(ctx *cli.Context, args []string) {
			wd, err := os.Getwd()
			if checkErr(ctx, err) {
				return
			}
			io.WriteString(ctx, fmt.Sprintln(wd))
		},
	}
}

func writeCmd() *cli.Command {
	return &cli.Command{
		Usage: "write <filepath> [text]...",
		Args:  cli.MinArgs(1),
		Run: func(ctx *cli.Context, args []string) {
			err := os.WriteFile(absPath(args[0]), []byte(strings.Join(args[1:], " ")), 0644)
			if checkErr(ctx, err) {
				return
			}
		},
	}
}

func printEnvCmd() *cli.Command {
	return &cli.Command{
		Usage: "env",
		Args:  cli.ExactArgs(0),
		Run: func(ctx *cli.Context, args []string) {
			for _, kvp := range os.Environ() {
				fmt.Fprintln(ctx, kvp)
			}
		},
	}
}

func exportCmd() *cli.Command {
	var remove bool

	cmd := &cli.Command{
		Usage: "export [-remove] <NAME[=VALUE]>...",
		Args:  cli.MinArgs(1),
		Short: "Set or remove environment variables.",
		Run: func(ctx *cli.Context, args []string) {
			for i, arg := range args {
				name, value, _ := strings.Cut(arg, "=")
				if name == "" {
					io.WriteString(ctx, fmt.Sprintf("invalid argument (%d): missing variable name", i))
					return
				}
				if remove {
					os.Unsetenv(name)
				} else {
					os.Setenv(name, value)
				}
			}
		},
	}

	// weird but it works
	cmd.Flags().BoolVar(&remove, "remove", false, "Remove an environment variable")
	return cmd
}

// todo: port from afero to engine/fs so watchfs works
// --
// var watches = make(map[string]*watchfs.Watch)
// func watchCmd() *cli.Command {
// 	cmd := &cli.Command{
// 		Usage: "watch <path>", // todo add -r
// 		Args:  cli.ExactArgs(1),
// 		Run: func(ctx *cli.Context, args []string) {
// 			var recursive bool
// 			var path string
// 			if args[0] == "-r" {
// 				recursive = true
// 				path = args[1]
// 			} else {
// 				path = args[0]
// 			}
// 			if _, exists := watches[path]; exists {
// 				return
// 			}
// 			w, err := watchfs.WatchFile(fs, path, &watchfs.Config{
// 				Recursive: recursive,
// 				Handler: func(e watchfs.Event) {
// 					js.Global().Get("console").Call("log", e.String())
// 				},
// 			})
// 			if err != nil {
// 				fmt.Fprintf(t, "%s\n", err)
// 				return
// 			}
// 			watches[args[0]] = w
// 		},
// 	}
// 	return cmd
// }
// func unwatchCmd() *cli.Command {
// 	cmd := &cli.Command{
// 		Usage: "unwatch <path>", // todo add -r
// 		Args:  cli.ExactArgs(1),
// 		Run: func(ctx *cli.Context, args []string) {
// 			w, exists := watches[args[0]]
// 			if !exists {
// 				return
// 			}
// 			w.Close()
// 			delete(watches, args[0])
// 			go func() {
// 				for e := range w.Iter() {
// 					js.Global().Get("console").Call("log", e.String())
// 				}
// 			}()
// 		},
// 	}
// 	return cmd
// }

// TODO: port these debug commands to cli.Commands like above
// "resetfs": func(t *term.Terminal, _ afero.Fs, args []string) {
// 	fs.Reset(nil)
// },
// "fsdata": func(t *term.Terminal, fs afero.Fs, args []string) {
// 	// watchFS := fs.(*watchfs.FS)
// 	// watched := GetUnexportedField(reflect.ValueOf(watchFS).Elem().FieldByIndex([]int{0}))
// 	cowFS := fs.(*afero.CopyOnWriteFs)
// 	layer := GetUnexportedField(reflect.ValueOf(cowFS).Elem().FieldByName("layer"))
// 	memFS := layer.(*afero.MemMapFs)
// 	data := GetUnexportedField(reflect.ValueOf(memFS).Elem().FieldByName("data"))
// 	fdata := data.(map[string]*mem.FileData)

// 	for name, fd := range fdata {
// 		memDir := GetUnexportedField(reflect.ValueOf(fd).Elem().FieldByName("memDir"))
// 		fmt.Printf("%s:\nFileData:%+v\nDirMap:%+v\n", name, *fd, memDir)
// 	}
// },

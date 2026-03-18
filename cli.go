package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitwisepossum/notch/todo"
)

func cmdLs(w io.Writer) error {
	names, err := todo.ListAll()
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Fprintln(w, name)
	}
	return nil
}

func cmdCat(w io.Writer, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: notch cat <list>")
	}
	list, err := todo.Load(args[0])
	if err != nil {
		return err
	}
	return todo.Write(w, list.Items)
}

func cmdAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: notch add <list> <item>")
	}
	name := args[0]
	text := strings.Join(args[1:], " ")
	list, err := todo.Load(name)
	if os.IsNotExist(err) {
		list = &todo.List{Name: name}
	} else if err != nil {
		return err
	}
	list.Add(nil, text)
	return todo.Save(list)
}

func cmdDone(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: notch done <list> <search>")
	}
	name := args[0]
	query := strings.Join(args[1:], " ")
	list, err := todo.Load(name)
	if err != nil {
		return err
	}
	results := list.Search(query)
	if len(results) == 0 {
		return fmt.Errorf("no item matching %q in %q", query, name)
	}
	r := results[0]
	if r.Item.Done {
		return fmt.Errorf("%q is already done", r.Item.Text)
	}
	list.Toggle(r.Path)
	return todo.Save(list)
}

func cmdRm(stdin io.Reader, args []string) error {
	force := false
	var name string
	for _, a := range args {
		switch a {
		case "-f", "--force":
			force = true
		default:
			if name != "" {
				return fmt.Errorf("usage: notch rm [-f] <list>")
			}
			name = a
		}
	}
	if name == "" {
		return fmt.Errorf("usage: notch rm [-f] <list>")
	}
	if !force {
		fmt.Fprintf(os.Stderr, "delete list %q? [y/N] ", name)
		var resp string
		fmt.Fscanln(stdin, &resp)
		if resp != "y" && resp != "Y" {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}
	}
	return todo.Delete(name)
}

func cmdVersion(w io.Writer) error {
	fmt.Fprintln(w, "notch v"+todo.Version)
	return nil
}

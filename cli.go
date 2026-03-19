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

func cmdVersion(w io.Writer) error {
	fmt.Fprintln(w, "notch v"+todo.Version)
	return nil
}

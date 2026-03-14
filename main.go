package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type TodoList struct {
	rdb *redis.Client
}

func NewTodoList() *TodoList {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is not set")
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &TodoList{rdb: rdb}
}

func (t *TodoList) Add(text string) {
	id, err := t.rdb.Incr(ctx, "todo:next_id").Result()
	if err != nil {
		return
	}

	key := taskKey(int(id))
	t.rdb.HSet(ctx, key, "text", text, "done", "0")

	t.rdb.RPush(ctx, "todo:ids", id)
	fmt.Printf("Task added with ID #%d\n", id)
}

func (t *TodoList) Complete(id int) {
	key := taskKey(id)

	n, err := t.rdb.HSet(ctx, key, "done", "1").Result()
	if err != nil || n == 0 {
		fmt.Println("Task not found")
		return
	}

	fmt.Println("Task completed!")
}

func (t *TodoList) Delete(id int) {
	key := taskKey(id)
	n, err := t.rdb.Del(ctx, key).Result()
	if err != nil || n == 0 {
		fmt.Println("Task not found")
		return
	}

	t.rdb.LRem(ctx, "todo:ids", 0, strconv.Itoa(id))
	fmt.Println("Task deleted!")
}

func (t *TodoList) List() {
	ids, err := t.rdb.LRange(ctx, "todo:ids", 0, -1).Result()
	if err != nil {
		fmt.Println("List is empty")
		return
	}

	for _, id := range ids {

		fields, err := t.rdb.HGetAll(ctx, taskKey(id)).Result()
		if err != nil || len(fields) == 0 {
			continue
		}

		status := "[ ]"
		if fields["done"] == "1" {
			status = "[x]"
		}
		fmt.Printf("%s #%s: %s\n", status, id, fields["text"])

	}
}

func taskKey[T any](id T) string {
	return fmt.Sprintf("todo:task:%v", id)
}

func main() {
	todo := NewTodoList()

	if err := todo.rdb.Ping(ctx).Err(); err != nil {
		log.Panicf("Redis unavalible: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== TODO + Redis ===")
	fmt.Println("add <text> | list | done <id> | del <id> | quit")

	for {
		fmt.Println("> ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(input, " ", 2)
		cmd := parts[0]

		switch cmd {
		case "add":
			if len(parts) < 2 {
				fmt.Println("enter the text of the task")
				continue
			}
			todo.Add(parts[1])
		case "list":
			todo.List()

		case "done", "del":
			if len(parts) < 2 {
				fmt.Println("enter the id of the task")
				continue
			}

			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("id should be a number")
			}

			if cmd == "del" {
				todo.Delete(id)
			} else {
				todo.Complete(id)
			}
		case "quit":
			return
		default:
			fmt.Println("unknown command")
		}
	}
}

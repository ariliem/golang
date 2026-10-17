package main

import (
	"bufio"
	"fmt"
	"os"
	"sis-ptoject/analyzer"
	"strings"
)

func main() {
	storage := analyzer.NewInMemoryStorage()
	service := analyzer.NewService(storage)

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Анализатор текста")
	fmt.Println("Ваш текст, введите exit для выхода:")

	for {
		fmt.Print("\nВведите текст > ")

		if !scanner.Scan() {
			break
		}

		input := scanner.Text()

		if strings.TrimSpace(strings.ToLower(input)) == "exit" {
			fmt.Println("Пока")
			break
		}

		task, err := service.CreateTask(input)
		if err != nil {
			fmt.Println("Ошибка валидации:", err)
			continue
		}
		fmt.Printf("Создана задача #%d [Статус: %s]\n", task.ID, task.State)

		processedTask, err := service.ProcessTask(task.ID)
		if err != nil {
			fmt.Println("Ошибка обработки:", err)
			continue
		}

		fmt.Printf("Задача #%d обработана [Статус: %s]\n", processedTask.ID, processedTask.State)
		fmt.Printf("Результат:\n")
		fmt.Printf("   - Слов: %d\n", processedTask.Result.WordCount)
		fmt.Printf("   - Символов: %d\n", processedTask.Result.CharCount)
		fmt.Printf("   - Гласных: %d\n", processedTask.Result.VowelCount)
		fmt.Printf("   - Самое длинное слово: %s\n", processedTask.Result.LongestWord)
	}
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"sis-ptoject/analyzer"
	"strings"
	"time"
)

func main() {
	storage := analyzer.NewInMemoryStorage()

	service := analyzer.NewServiceWithConfig(storage, 3, 5)
	service.Start()

	fmt.Println("=== Анализатор текста (Checkpoint 2: Worker Pool) ===")
	fmt.Println("Запущено 3 воркера, размер очереди: 5.")
	fmt.Println("Введите текст для асинхронной обработки (или 'exit' для выхода):")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nВведите текст > ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if strings.TrimSpace(strings.ToLower(input)) == "exit" {
			fmt.Println("Остановка воркеров и завершение оставшихся задач...")
			service.Shutdown()
			fmt.Println("Все воркеры остановлены. Пока!")
			break
		}

		task, err := service.CreateTask(input)
		if err != nil {
			fmt.Println("❌ Ошибка валидации:", err)
			continue
		}

		if err := service.SubmitTask(task.ID); err != nil {
			fmt.Println("❌ Ошибка отправки в очередь:", err)
			continue
		}
		fmt.Printf("📥 Задача #%d принята в очередь\n", task.ID)

		time.Sleep(50 * time.Millisecond)

		resTask, err := service.GetTask(task.ID)
		if err == nil && resTask.State == analyzer.StateCompleted {
			fmt.Printf("✅ Задача #%d обработана [Статус: %s]\n", resTask.ID, resTask.State)
			fmt.Printf("   Слов: %d | Символов: %d | Гласных: %d | Самое длинное слово: %s\n",
				resTask.Result.WordCount,
				resTask.Result.CharCount,
				resTask.Result.VowelCount,
				resTask.Result.LongestWord,
			)
		}
	}
}

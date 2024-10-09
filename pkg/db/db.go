package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func DbOpen() *sql.DB {
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = dbCheck()
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS scheduler (id INTEGER PRIMARY KEY AUTOINCREMENT, date TEXT, title TEXT, comment TEXT, repeat TEXT)")
	if err != nil {
		panic(err)
	}
	return db
}

// Проверяет наличие базы данных в директории выполнения приложения, создает БД если ее нет и возвращает путь до нее.
func dbCheck() string {
	appPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dbFile := filepath.Join(filepath.Dir(appPath), "scheduler.db")
	_, err = os.Stat(dbFile)

	if err != nil {
		os.Create("scheduler.db")
	}

	return dbFile
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	switch {
	case strings.HasPrefix(repeat, "d "):
		daysStr := strings.TrimPrefix(repeat, "d ")
		daysNum, err := strconv.Atoi(daysStr)
		if err != nil {
			return "", fmt.Errorf("неверный формат: %s ; %v", repeat, err)
		}

		if daysNum >= 400 {
			return "", fmt.Errorf("перенос задачи на 400 и более дней: %s;", repeat)
		}

		taskDate, err := time.Parse("20060102", date)

		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", date)
		}

		taskDate = taskDate.AddDate(0, 0, daysNum) // для учета пограничных вариантов

		for taskDate.Before(now) {
			taskDate = taskDate.AddDate(0, 0, daysNum)
		}

		return taskDate.Format("20060102"), nil

	case repeat == "y":
		taskDate, err := time.Parse("20060102", date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", date)
		}

		taskDate = taskDate.AddDate(1, 0, 0) // для учета пограничных вариантов
		for taskDate.Before(now) {
			taskDate = taskDate.AddDate(1, 0, 0)
		}
		return taskDate.Format("20060102"), nil

	case strings.HasPrefix(repeat, "w "):
		weekdaysStr := strings.Split(strings.TrimPrefix(repeat, "w "), ",")
		weekdaysInt := make([]int, len(weekdaysStr))
		for i := range weekdaysStr {
			num, err := strconv.Atoi(weekdaysStr[i])
			if err != nil {
				return "", fmt.Errorf("неверный формат: %s ; %v", repeat, err)
			}
			if num > 7 || num < 0 {
				return "", fmt.Errorf("неверный формат: %s ; %v", repeat, err)
			}
			if num == 7 {
				num = 0
			}
			weekdaysInt[i] = num
		}
		taskDate, err := time.Parse("20060102", date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", date)
		}

		if !taskDate.After(now) {
			taskDate = now.AddDate(0, 0, 1)
		}

		found := false
		for {
			for _, day := range weekdaysInt {

				if int(taskDate.Weekday()) == day {
					found = true
					break
				}

			}
			if found {
				break
			}
			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format("20060102"), nil

	case strings.HasPrefix(repeat, "m "):
		splitted := strings.Split(repeat, " ")
		fmt.Println(splitted)
		if len(splitted) > 3 || len(splitted) < 2 {
			return "", fmt.Errorf("неверный формат: %s;", repeat)
		}

		daysStr := strings.Split(splitted[1], ",")
		var daysNum []int

		for i := range daysStr {
			dayNum, err := strconv.Atoi(daysStr[i])
			if err != nil || dayNum > 31 || dayNum == 0 || dayNum < -2 {
				return "", fmt.Errorf("неверный формат: %s;", repeat)
			}

			daysNum = append(daysNum, dayNum)
		}

		sort.Slice(daysNum, func(i, j int) bool {
			return daysNum[i] < daysNum[j]
		})

		taskDate, err := time.Parse("20060102", date)
		if err != nil {
			return "", fmt.Errorf("ошибка при считывании даты: %s", date)
		}

		if !taskDate.After(now) {
			taskDate = now.AddDate(0, 0, 1)
		}
		found := false

		taskDate = checkFirstMonth(daysNum, taskDate)

		if len(splitted) == 3 {

			monthsStr := strings.Split(splitted[2], ",")
			var months []int

			for i := range monthsStr {
				mthNum, err := strconv.Atoi(monthsStr[i])
				if err != nil || mthNum > 12 || mthNum < 1 {
					return "", fmt.Errorf("неверный формат: %s;", repeat)
				}

				months = append(months, mthNum)
			}

			sort.Slice(months, func(i, j int) bool {
				return months[i] < months[j]
			})

			for {
				for _, v := range months {
					if int(taskDate.Month()) == v {
						found = true
						break
					}
				}

				if found {
					break
				}

				taskDate = taskDate.AddDate(0, 1, 0)
			}
			//
		}

		found = false
		for {
			for _, v := range daysNum {
				if v < 0 {
					v = v + 1
					lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
					v = lastDay.AddDate(0, 0, v).Day()
				}

				if taskDate.Day() == v {
					found = true
					break
				}
			}

			if found {
				break
			}

			taskDate = taskDate.AddDate(0, 0, 1)
		}
		return taskDate.Format("20060102"), nil

	default:
		return "", fmt.Errorf("неверный формат поля 'repeat': %s", repeat)
	}
}

func checkFirstMonth(daysNum []int, taskDate time.Time) time.Time {
	startMonth := taskDate.Month()
	found := false
	for {
		for _, v := range daysNum {
			if v < 0 {
				v = v + 1
				lastDay := time.Date(taskDate.Year(), taskDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
				v = lastDay.AddDate(0, 0, v).Day()
			}

			if taskDate.Day() == v {
				found = true
				break
			}
		}

		if found {
			break
		}

		taskDate = taskDate.AddDate(0, 0, 1)
		if taskDate.Month() != startMonth {
			return taskDate
		}
	}
	return taskDate
}

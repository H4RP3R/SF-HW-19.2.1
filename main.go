package main

import (
	"fmt"
	"log"
	"sync"
)

// Число сообщений от источника
const messagesAmountPerGoroutine int = 5

// Функция разуплотнения каналов
func demultiplexingFunc(done chan struct{}, dataSourceChan chan int, amount int) []chan int {
	var output = make([]chan int, amount)
	// Обратите внимание: вышеприведённая команда инициализирует слайс,
	// но не инициализирует каждый его элемент. Каждый элемент будет
	// представлять так называемое нулевое значение
	// для данного типа.
	// Так как тип у нас канала — ссылочный, то все элементы
	// будут
	// равны nil
	for i := range output {
		output[i] = make(chan int)
	}
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			// При поступлении сообщения в канал-источник
			// отправляем его в каждый из каналов-потребителей
			for v := range dataSourceChan {
				for _, c := range output {
					c <- v
				}
			}
		}()
		wg.Wait()
		// После завершения посылки сообщений в основной
		// канал-источник
		// данных
		// закрываем все каналы-потребители
		close(done)
	}()
	return output
}

// Функция уплотнения каналов
func multiplexingFunc(done chan struct{}, channels ...chan int) <-chan int {
	var wg sync.WaitGroup
	// Общий канал, в который будут попадать сообщения от всех
	// источников
	// Именно его мы и вернём из этой функции для употребления
	// внешним кодом
	multiplexedChan := make(chan int)
	multiplex := func(c <-chan int) {
		defer func() {
			log.Println("consumer stopped")
			wg.Done()
		}()
		// Если поступило сообщение из одного из
		// каналов-источников,
		// перенаправляем его в общий канал
		for {
			select {
			case msg, ok := <-c:
				if !ok {
					return
				}
				multiplexedChan <- msg
			case <-done:
				return
			}
		}
	}
	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}
	// Запускаем горутину, которая закроет канал после того,
	// как в закрывающий канал поступит сигнал о прекращении
	// работы всех
	go func() {
		wg.Wait()
		close(multiplexedChan)
	}()
	return multiplexedChan
}

func main() {
	var done = make(chan struct{})
	// Горутина — источник данных
	// Функция создаёт свой собственный канал
	// и посылает в него пять сообщений
	startDataSource := func() chan int {
		c := make(chan int)
		go func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 1; i <= messagesAmountPerGoroutine; i++ {
					c <- i
				}
			}()
			wg.Wait()
			close(c)
		}()
		return c
	}
	// Запускаем источник данных и уплотняем каналы
	var consumers []chan int = demultiplexingFunc(done, startDataSource(), 5)
	c := multiplexingFunc(done, consumers...)
	// Централизованно получаем сообщения от всех нужных нам
	// источников
	// данных
	for data := range c {
		fmt.Println(data)
	}
}

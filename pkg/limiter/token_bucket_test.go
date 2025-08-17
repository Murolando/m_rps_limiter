package limiter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cristiano murolando"))
}

func TestLimiterHandler(t *testing.T) {
	tests := []struct {
		name            string
		limit           int32
		requests        int
		interval        time.Duration
		expectedPassed  int
		expectedBlocked int
	}{
		{
			name:            "Базовое ограничение - 2 запроса",
			limit:           2,
			requests:        5,
			interval:        time.Millisecond * 10,
			expectedPassed:  2,
			expectedBlocked: 3,
		},
		{
			name:            "Один запрос в секунду",
			limit:           1,
			requests:        3,
			interval:        time.Millisecond * 10,
			expectedPassed:  1,
			expectedBlocked: 2,
		},
		{
			name:            "Высокий лимит",
			limit:           10,
			requests:        5,
			interval:        time.Millisecond * 10,
			expectedPassed:  5,
			expectedBlocked: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тикер с коротким интервалом для тестов
			ticker := time.NewTicker(time.Millisecond * 50)
			defer ticker.Stop()

			// Создаем лимитер
			limiter, err := NewLimiter(
				tt.limit,
				TokenBucket,
				ticker,
			)
			if err != nil {
				t.Fatalf("Ошибка создания лимитера: %v", err)
			}
			defer limiter.Stop()

			// Создаем хендлер с лимитером
			handler := limiter.Limit(http.HandlerFunc(testHandler))

			var passed, blocked int

			// Отправляем запросы
			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				switch w.Code {
				case http.StatusOK:
					passed++
				case http.StatusTooManyRequests:
					blocked++
				}

				time.Sleep(tt.interval)
			}

			// Проверяем результаты
			if passed != tt.expectedPassed {
				t.Errorf("Ожидалось %d успешных запросов, получено %d",
					tt.expectedPassed, passed)
			}

			if blocked != tt.expectedBlocked {
				t.Errorf("Ожидалось %d заблокированных запросов, получено %d",
					tt.expectedBlocked, blocked)
			}
		})
	}
}

func TestLimiterWithTokenRefill(t *testing.T) {
	// Тест восстановления токенов
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	limiter, err := NewLimiter(
		2, // лимит 2 токена
		TokenBucket,
		ticker,
	)
	if err != nil {
		t.Fatalf("Ошибка создания лимитера: %v", err)
	}
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(testHandler))

	// Исчерпываем токены
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Запрос %d должен был пройти", i+1)
		}
	}

	// Следующий запрос должен быть заблокирован
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Error("Запрос должен был быть заблокирован")
	}

	// Ждем восстановления токенов
	time.Sleep(time.Millisecond * 250)

	// Теперь запрос должен пройти
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Error("Запрос должен был пройти после восстановления токенов")
	}
}

func TestLimiterErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		limit       int32
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Нулевой лимит",
			limit:       0,
			expectError: true,
			errorMsg:    "limit must be greater than 0",
		},
		{
			name:        "Отрицательный лимит",
			limit:       -1,
			expectError: true,
			errorMsg:    "limit must be greater than 0",
		},
		{
			name:        "Валидный лимит",
			limit:       5,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter, err := NewLimiter(
				tt.limit,
				TokenBucket,
				nil,
			)

			if tt.expectError {
				if err == nil {
					t.Error("Ожидалась ошибка, но её не было")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Ожидалось сообщение об ошибке '%s', получено '%s'",
						tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Неожиданная ошибка: %v", err)
				}
				if limiter != nil {
					limiter.Stop()
				}
			}
		})
	}
}

func TestTokenBucketConcurrency(t *testing.T) {
	// Тест на конкурентность
	ticker := time.NewTicker(time.Millisecond * 50)
	defer ticker.Stop()

	limiter, err := NewLimiter(
		5, // лимит 5 токенов
		TokenBucket,
		ticker,
	)
	if err != nil {
		t.Fatalf("Ошибка создания лимитера: %v", err)
	}
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(testHandler))

	// Запускаем несколько горутин одновременно
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	var passed, blocked int
	for i := 0; i < 10; i++ {
		code := <-results
		switch code {
		case http.StatusOK:
			passed++
		case http.StatusTooManyRequests:
			blocked++
		}
	}

	// Должно пройти не больше 5 запросов (лимит)
	if passed > 5 {
		t.Errorf("Прошло слишком много запросов: %d, ожидалось максимум 5", passed)
	}

	// Общее количество должно быть 10
	if passed+blocked != 10 {
		t.Errorf("Общее количество запросов неверно: %d + %d != 10", passed, blocked)
	}
}

// Бенчмарк для проверки производительности
func BenchmarkLimiterHandler(b *testing.B) {
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()

	limiter, err := NewLimiter(
		1000, // высокий лимит для бенчмарка
		TokenBucket,
		ticker,
	)
	if err != nil {
		b.Fatalf("Ошибка создания лимитера: %v", err)
	}
	defer limiter.Stop()

	handler := limiter.Limit(http.HandlerFunc(testHandler))
	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

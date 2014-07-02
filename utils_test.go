package gorm

import "testing"

func TestToSnakeWithCapWord(t *testing.T) {
    in, out := "BlockIOCount", "block_io_count"
    
    if s := toSnake(in); s != out {
        t.Errorf("toSnake(%v) = %v, want %v", in, s, out )
    }
}

func TestToSnakeWithCapWordAtEnd(t *testing.T) {
    in, out := "BlockIO", "block_io"
    
    if s := toSnake(in); s != out {
        t.Errorf("toSnake(%v) = %v, want %v", in, s, out )
    }
}

func BenchmarkToSnake(b *testing.B) {
    in := "BlockIOCount"
    
    for i := 0; i < b.N; i++ {
        toSnakeBody(in)
    }

}

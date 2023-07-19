package itertools_test

import (
	"fmt"
	"testing"

	"github.com/neild/itertools"
)

func ExampleCount() {
	for i := range itertools.Count(1) {
		fmt.Println(i)
		if i >= 3 {
			break
		}
	}
	// Output:
	// 1
	// 2
	// 3
}

func ExampleCountBy() {
	for i := range itertools.CountBy(0, 2) {
		fmt.Println(i)
		if i >= 4 {
			break
		}
	}
	// Output:
	// 0
	// 2
	// 4
}

func ExampleRepeat() {
	i := 0
	for v := range itertools.Repeat(7) {
		fmt.Println(v)
		i++
		if i >= 3 {
			break
		}
	}
	// Output:
	// 7
	// 7
	// 7
}

func ExampleRepeatN() {
	for v := range itertools.RepeatN("value", 3) {
		fmt.Println(v)
	}
	// Output:
	// value
	// value
	// value
}

func ExampleAccumulate() {
	for v := range itertools.Accumulate(itertools.Range(1, 6), func(a, b int) int {
		return a + b
	}) {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 3
	// 6
	// 10
	// 15
}

func ExampleChain() {
	for v := range itertools.Chain(
		itertools.FromSlice([]byte("abc")),
		itertools.FromSlice([]byte("def")),
	) {
		fmt.Printf("%c", v)
	}
	fmt.Println()
	// Output:
	// abcdef
}

func ExampleChainFromIter() {
	for v := range itertools.ChainFromIter(
		itertools.FromSlice([]itertools.Iter[byte]{
			itertools.FromSlice([]byte("abc")),
			itertools.FromSlice([]byte("def")),
		}),
	) {
		fmt.Printf("%c", v)
	}
	fmt.Println()
	// Output:
	// abcdef
}

func ExampleCompress() {
	for v := range itertools.Compress(
		itertools.FromSlice([]string{"A", "B", "C", "D", "E", "F"}),
		itertools.FromSlice([]bool{true, false, true, false, true, true}),
	) {
		fmt.Println(v)
	}
	// Output:
	// A
	// C
	// E
	// F
}

func ExampleDropWhile() {
	for v := range itertools.DropWhile(
		itertools.FromSlice([]int{1, 4, 6, 4, 1}),
		func(x int) bool {
			return x < 5
		},
	) {
		fmt.Println(v)
	}
	// Output:
	// 6
	// 4
	// 1
}

func ExampleFilterFalse() {
	for v := range itertools.FilterFalse(
		itertools.Range(0, 10),
		func(x int) bool {
			return x%2 != 0
		},
	) {
		fmt.Println(v)
	}
	// Output:
	// 0
	// 2
	// 4
	// 6
	// 8
}

func ExampleGroupBy_Keys() {
	for k, _ := range itertools.GroupBy(
		itertools.FromSlice([]byte("AAAABBBCCDAABBB")),
		func(x byte) string {
			return string([]byte{x})
		},
	) {
		fmt.Println(k)
	}
	// Output:
	// A
	// B
	// C
	// D
	// A
	// B
}

func ExampleGroupBy_Groups() {
	for k, g := range itertools.GroupBy(
		itertools.FromSlice([]byte("AAAABBBCCDAABBB")),
		func(x byte) string {
			return string([]byte{x})
		},
	) {
		fmt.Print(k, ": ")
		for v := range g {
			fmt.Print(string([]byte{v}))
		}
		fmt.Println()
	}
	// Output:
	// A: AAAA
	// B: BBB
	// C: CC
	// D: D
	// A: AA
	// B: BBB
}

func ExampleSlice() {
	for v := range itertools.Slice(itertools.Count(0), 2, 5) {
		fmt.Println(v)
	}
	// Output:
	// 2
	// 3
	// 4
}

func ExamplePairwise() {
	for a, b := range itertools.Pairwise(itertools.Range(0, 4)) {
		fmt.Println(a, b)
	}
	// Output:
	// 0 1
	// 1 2
	// 2 3
}

func ExampleTakeWhile() {
	for v := range itertools.TakeWhile(
		itertools.FromSlice([]int{1, 4, 6, 4, 1}),
		func(x int) bool {
			return x < 5
		},
	) {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 4
}

func ExampleTee() {
	iters := itertools.Tee(itertools.Range(0, 3), 2)
	for _, iter := range iters {
		for v := range iter {
			fmt.Println(v)
		}
	}
	// Output:
	// 0
	// 1
	// 2
	// 0
	// 1
	// 2
}

func TestTee(t *testing.T) {
	iters := itertools.Tee(itertools.Range(0, 1000), 4)
	var pulls [4]func() (int, bool)
	for i, iter := range iters {
		var stop func()
		pulls[i], stop = itertools.Pull(iter)
		defer stop()
	}
	var seqs [4][]int
	running := 4
	for running > 0 {
		for i, next := range pulls {
			if next == nil {
				continue
			}
			for range i + 1 {
				v, ok := next()
				if !ok {
					pulls[i] = nil
					running--
					break
				}
				seqs[i] = append(seqs[i], v)
			}
		}
	}
	fmt.Println(seqs)
}

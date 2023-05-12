# go-sudoku-solver
An Optimized Sudoku Solver and Generator written in Go, with a lighting fast speed.
Using optimized backtracking: reduce the decision space for each cell before backtracking, even solved the puzzle already without even backtracking.

## Example Input

A very hard Sudoku problem for normal brute-force algorithms

```
.......1.
4........
.2.......
....5.4.7
..8...3..
..1.9....
3..4..2..
.5.1.....
...8.6...
```

## Example Output

```
...|...|.1.
4..|...|...
.2.|...|...
___ ___ ___
...|.5.|4.7
..8|...|3..
..1|.9.|...
___ ___ ___
3..|4..|2..
.5.|1..|...
...|8.6|...

693|784|512
487|512|936
125|963|874
___ ___ ___
932|651|487
568|247|391
741|398|625
___ ___ ___
319|475|268
856|129|743
274|836|159

Recursive count: 34031490
```

# sample feature vector derivation

`internal/policy.Features` maps a grid to `[clues, PropagationFill, unitMin, unitMax, unitSum/27]` over the 27 units, 9 rows, 9 columns, 9 boxes. For the classic sample fixture `internal/puzzle.Sample81` the vector is:

```text
[17 64 0 4 1.8888888888888888]
```

Every component below was derived by hand from the fixture string, then confirmed by a scratch program in `playground/features_check` that calls the real `Features`, `PropagationFill`, and `Propagate`.

## the sample

```text
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

17 clues, hence `clues = 17` and 64 empty cells.

## unit census

Rows, with placed digits:

| unit | clues | digits |
|---|---|---|
| row 1 | 1 | 1 |
| row 2 | 1 | 4 |
| row 3 | 1 | 2 |
| row 4 | 3 | 5 4 7 |
| row 5 | 2 | 8 3 |
| row 6 | 2 | 1 9 |
| row 7 | 3 | 3 4 2 |
| row 8 | 2 | 5 1 |
| row 9 | 2 | 8 6 |

Columns:

| unit | clues | digits |
|---|---|---|
| col 1 | 2 | 4 3 |
| col 2 | 2 | 2 5 |
| col 3 | 2 | 8 1 |
| col 4 | 3 | 4 1 8 |
| col 5 | 2 | 5 9 |
| col 6 | 1 | 6 |
| col 7 | 3 | 4 3 2 |
| col 8 | 1 | 1 |
| col 9 | 1 | 7 |

Boxes:

| unit | clues | digits |
|---|---|---|
| box 1 | 2 | 4 2 |
| box 2 | 0 | none |
| box 3 | 1 | 1 |
| box 4 | 2 | 8 1 |
| box 5 | 2 | 5 9 |
| box 6 | 3 | 4 7 3 |
| box 7 | 2 | 3 5 |
| box 8 | 4 | 4 1 8 6 |
| box 9 | 1 | 2 |

An earlier draft of this census recorded col 6 with 3 clues and col 7 with 1. That was a transcription error, corrected here: col 6 holds only r9c6 = 6, and col 7 holds r4c7 = 4, r5c7 = 3, r7c7 = 2. The vector above uses the corrected counts.

From the census: `unitMin = 0` because box 2 is empty, `unitMax = 4` in box 8, and `unitSum = 51` because each of the 17 clues belongs to exactly 3 units, 3 x 17 = 51, so `unitSum/27 = 51/27 = 1.8888888888888888`, the mean clue load of a unit.

## propagationFill = 64

The sample solves by singles alone, no search branch ever fires. All 64 empty cells fill through naked or hidden singles, so `PropagationFill` returns 64 and `Propagate` finishes the grid in exactly 64 steps, which is the pinned step count for the classic sample. The first single of each kind, worked out:

- r7c5 = 7, a naked single. Row 7 holds {3, 4, 2}, col 5 holds {5, 9}, box 8 holds {4, 1, 8, 6}. The union is {1, 2, 3, 4, 5, 6, 8, 9}, missing only 7. The fixture solution row 7 reads `319475268`, position 5 is 7.
- r4c6 = 1, a hidden single for row 4. Row 4 is `[. . . . 5 . 4 . 7]` and digit 1 has nowhere else in it: cols 1 to 3 sit in box 4, which already has 1 at r6c3, col 4 has 1 at r8c4, col 8 has 1 at r1c8, and col 9 is the given 7. That leaves col 6, whose column holds only 6 at r9c6 and whose box 5 holds {5, 9}, so 1 fits. The fixture solution row 4 reads `932651487`, position 6 is 1.

## machine confirmation

The scratch check printed `features: [17 64 0 4 1.8888888888888888]`, `fill: 64`, the same three census vectors, `propagate solved: true steps: 64`, and the propagate output equal to `SampleSolution81` byte for byte.

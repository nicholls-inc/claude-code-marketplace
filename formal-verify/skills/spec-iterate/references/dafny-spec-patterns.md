# Dafny Specification Patterns Reference

## Basic Function vs Method

```dafny
// Pure function (no side effects, can be used in specs)
function Abs(x: int): int {
  if x < 0 then -x else x
}

// Method (can have side effects, uses requires/ensures)
method ComputeAbs(x: int) returns (result: int)
  ensures result == Abs(x)
{
  if x < 0 { result := -x; } else { result := x; }
}
```

## Array Patterns

### Quantifiers over arrays
```dafny
// All elements satisfy a property
ensures forall i :: 0 <= i < a.Length ==> a[i] >= 0

// At least one element satisfies a property
ensures exists i :: 0 <= i < a.Length && a[i] == target

// Array is sorted
predicate Sorted(a: array<int>)
  reads a
{
  forall i, j :: 0 <= i < j < a.Length ==> a[i] <= a[j]
}
```

### Array modification
```dafny
method Swap(a: array<int>, i: int, j: int)
  requires 0 <= i < a.Length && 0 <= j < a.Length
  modifies a
  ensures a[i] == old(a[j]) && a[j] == old(a[i])
  ensures forall k :: 0 <= k < a.Length && k != i && k != j ==> a[k] == old(a[k])
```

## Sequence Patterns

```dafny
// Sequence concatenation and slicing
ensures result == s[..i] + s[i+1..]

// Multiset equality (permutation)
ensures multiset(result) == multiset(input)

// Sequence sum
function SumSeq(s: seq<int>): int {
  if |s| == 0 then 0
  else s[0] + SumSeq(s[1..])
}
```

## Loop Invariants

```dafny
method LinearSearch(a: array<int>, key: int) returns (index: int)
  ensures index >= 0 ==> index < a.Length && a[index] == key
  ensures index < 0 ==> forall i :: 0 <= i < a.Length ==> a[i] != key
{
  index := 0;
  while index < a.Length
    invariant 0 <= index <= a.Length
    invariant forall i :: 0 <= i < index ==> a[i] != key
  {
    if a[index] == key { return; }
    index := index + 1;
  }
  index := -1;
}
```

## Termination (Decreases Clauses)

```dafny
// Simple recursion
function Factorial(n: nat): nat
  decreases n
{
  if n == 0 then 1 else n * Factorial(n - 1)
}

// Mutual recursion
function IsEven(n: nat): bool
  decreases n
{
  if n == 0 then true else IsOdd(n - 1)
}

function IsOdd(n: nat): bool
  decreases n
{
  if n == 0 then false else IsEven(n - 1)
}

// Lexicographic termination
method NestedLoops(m: nat, n: nat)
  decreases m, n
```

## Ghost State and Lemmas

```dafny
// Ghost variable (exists only for verification, not compiled)
ghost var proof_witness: int;

// Lemma (proof obligation, not compiled)
lemma DistributiveProperty(a: int, b: int, c: int)
  ensures a * (b + c) == a * b + a * c
{}

// Calc blocks for structured proofs
lemma SumCommutative(a: int, b: int)
  ensures a + b == b + a
{
  calc {
    a + b;
    == b + a;
  }
}
```

## Predicates and Subset Types

```dafny
// Named predicate for readability
predicate IsPositive(x: int) { x > 0 }

// Subset type (refinement type)
type Positive = x: int | x > 0 witness 1

// Newtype with constraint
newtype Percentage = x: int | 0 <= x <= 100
```

## Datatypes (Algebraic Types)

```dafny
datatype Option<T> = None | Some(value: T)

datatype List<T> = Nil | Cons(head: T, tail: List<T>)

function Length<T>(l: List<T>): nat {
  match l
  case Nil => 0
  case Cons(_, tail) => 1 + Length(tail)
}
```

## Common Postcondition Patterns

```dafny
// Result is in bounds
ensures 0 <= result < a.Length

// Result preserves structure
ensures |result| == |input|

// Result is a permutation
ensures multiset(result[..]) == multiset(old(a[..]))

// Idempotence
ensures f(f(x)) == f(x)

// Monotonicity
ensures x <= y ==> f(x) <= f(y)
```

## Extern Methods (for IO / FFI)

```dafny
// Declare an external method (not verified, implemented in target language)
method {:extern} ReadLine() returns (s: string)

method {:extern} PrintLine(s: string)
```

These cannot be formally verified — they are trust boundaries.

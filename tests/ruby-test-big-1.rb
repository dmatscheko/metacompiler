# Ruby subset - big self test 1: DATA STRUCTURES AND ALGORITHMS.
# Sorting (bubble / selection / insertion / quicksort / merge sort), binary search,
# a Stack / Queue / singly linked list, gcd / lcm, and a frequency table - all built
# from the implemented subset only (arrays with [] get/set, << , .push/.pop/.each/
# .to_a/.size, hashes, blocks, classes with @ivars, recursion). It counts failed
# checks and ends with exit(fails), so exit 0 means the interpreter and the LLVM-IR
# compiler produced identical, correct results.

fails = 0

def check(name, got, want)
  if got != want
    puts "FAIL #{name}: got #{got} want #{want}"
    fails = fails + 1
  end
end

# Structural array equality (== on arrays is identity in this subset, so compare
# element by element).
def arr_eq(a, b)
  na = a.size
  return false if na != b.size
  i = 0
  while i < na
    return false if a[i] != b[i]
    i += 1
  end
  true
end

# Append every element of src onto dst (there is no array + array in the subset).
def concat_into(dst, src)
  src.each { |x| dst << x }
  dst
end

# A shallow copy so the sorts never mutate the caller's array.
def copy_of(arr)
  out = []
  arr.each { |x| out << x }
  out
end

# ----- sorting -----

def bubble_sort(input)
  arr = copy_of(input)
  n = arr.size
  i = 0
  while i < n
    j = 0
    while j < n - 1 - i
      if arr[j] > arr[j + 1]
        tmp = arr[j]
        arr[j] = arr[j + 1]
        arr[j + 1] = tmp
      end
      j += 1
    end
    i += 1
  end
  arr
end

def selection_sort(input)
  arr = copy_of(input)
  n = arr.size
  i = 0
  while i < n
    lo = i
    j = i + 1
    while j < n
      lo = j if arr[j] < arr[lo]
      j += 1
    end
    if lo != i
      tmp = arr[i]
      arr[i] = arr[lo]
      arr[lo] = tmp
    end
    i += 1
  end
  arr
end

def insertion_sort(input)
  arr = copy_of(input)
  i = 1
  while i < arr.size
    key = arr[i]
    j = i - 1
    while j >= 0 && arr[j] > key
      arr[j + 1] = arr[j]
      j -= 1
    end
    arr[j + 1] = key
    i += 1
  end
  arr
end

def quicksort(arr)
  return arr if arr.size <= 1
  pivot = arr[0]
  less = []
  more = []
  i = 1
  while i < arr.size
    if arr[i] < pivot
      less << arr[i]
    else
      more << arr[i]
    end
    i += 1
  end
  result = []
  concat_into(result, quicksort(less))
  result << pivot
  concat_into(result, quicksort(more))
  result
end

def merge(a, b)
  out = []
  i = 0
  j = 0
  while i < a.size && j < b.size
    if a[i] <= b[j]
      out << a[i]
      i += 1
    else
      out << b[j]
      j += 1
    end
  end
  while i < a.size
    out << a[i]
    i += 1
  end
  while j < b.size
    out << b[j]
    j += 1
  end
  out
end

def merge_sort(arr)
  return arr if arr.size <= 1
  mid = arr.size / 2
  left = []
  right = []
  i = 0
  while i < arr.size
    if i < mid
      left << arr[i]
    else
      right << arr[i]
    end
    i += 1
  end
  merge(merge_sort(left), merge_sort(right))
end

data = [5, 2, 9, 1, 5, 6, 3, 8, 3, 7, 0, 4]
expected = [0, 1, 2, 3, 3, 4, 5, 5, 6, 7, 8, 9]

check("bubble sort", arr_eq(bubble_sort(data), expected), true)
check("selection sort", arr_eq(selection_sort(data), expected), true)
check("insertion sort", arr_eq(insertion_sort(data), expected), true)
check("quicksort", arr_eq(quicksort(data), expected), true)
check("merge sort", arr_eq(merge_sort(data), expected), true)
# the original array must be untouched by the copying sorts
check("sort is pure", data[0], 5)
# an already sorted and a reversed input still come out sorted
check("sort sorted", arr_eq(quicksort([1, 2, 3, 4, 5]), [1, 2, 3, 4, 5]), true)
check("sort reversed", arr_eq(merge_sort([5, 4, 3, 2, 1]), [1, 2, 3, 4, 5]), true)
check("sort singleton", arr_eq(bubble_sort([42]), [42]), true)

# ----- binary search over a sorted array -----

def binary_search(arr, target)
  lo = 0
  hi = arr.size - 1
  while lo <= hi
    mid = (lo + hi) / 2
    if arr[mid] == target
      return mid
    elsif arr[mid] < target
      lo = mid + 1
    else
      hi = mid - 1
    end
  end
  return -1
end

sorted = [1, 3, 5, 7, 9, 11, 13, 15]
check("bsearch first", binary_search(sorted, 1), 0)
check("bsearch last", binary_search(sorted, 15), 7)
check("bsearch mid", binary_search(sorted, 7), 3)
check("bsearch miss low", binary_search(sorted, 0), -1)
check("bsearch miss gap", binary_search(sorted, 8), -1)
check("bsearch miss high", binary_search(sorted, 100), -1)

# ----- Stack (LIFO) -----

class Stack
  def initialize
    @items = []
  end
  def push(x)
    @items << x
    self
  end
  def pop
    @items.pop
  end
  def peek
    @items.last
  end
  def size
    @items.size
  end
  def empty?
    @items.size == 0
  end
end

st = Stack.new
check("stack empty", st.empty?, true)
st.push(1).push(2).push(3)
check("stack size", st.size(), 3)
check("stack peek", st.peek, 3)
check("stack pop", st.pop, 3)
check("stack pop2", st.pop, 2)
check("stack not empty", st.empty?, false)
check("stack pop3", st.pop, 1)
check("stack empty again", st.empty?, true)
check("stack pop empty", st.pop, nil)

# a stack reverses a sequence
def reverse_via_stack(arr)
  s = Stack.new
  arr.each { |x| s.push(x) }
  out = []
  until s.empty?
    out << s.pop
  end
  out
end
check("stack reverse", arr_eq(reverse_via_stack([1, 2, 3, 4]), [4, 3, 2, 1]), true)

# ----- Queue (FIFO) with a moving head index -----

class Queue
  def initialize
    @items = []
    @head = 0
  end
  def enqueue(x)
    @items << x
    self
  end
  def dequeue
    return nil if @head >= @items.size
    v = @items[@head]
    @head += 1
    v
  end
  def size
    @items.size - @head
  end
  def empty?
    self.size() == 0
  end
end

q = Queue.new
q.enqueue(10).enqueue(20).enqueue(30)
check("queue size", q.size(), 3)
check("queue deq1", q.dequeue, 10)
check("queue deq2", q.dequeue, 20)
q.enqueue(40)
check("queue size after mix", q.size(), 2)
check("queue deq3", q.dequeue, 30)
check("queue deq4", q.dequeue, 40)
check("queue empty", q.empty?, true)
check("queue deq empty", q.dequeue, nil)

# ----- singly linked list built from hashes -----

class LinkedList
  def initialize
    @head = nil
    @count = 0
  end
  def prepend(x)
    @head = {"val" => x, "next" => @head}
    @count += 1
    self
  end
  def length
    @count
  end
  def to_array
    out = []
    cur = @head
    until cur == nil
      out << cur["val"]
      cur = cur["next"]
    end
    out
  end
  def sum
    total = 0
    cur = @head
    until cur == nil
      total += cur["val"]
      cur = cur["next"]
    end
    total
  end
  def contains?(x)
    cur = @head
    until cur == nil
      return true if cur["val"] == x
      cur = cur["next"]
    end
    false
  end
end

list = LinkedList.new
list.prepend(3).prepend(2).prepend(1)
check("list length", list.length(), 3)
check("list order", arr_eq(list.to_array, [1, 2, 3]), true)
check("list sum", list.sum, 6)
check("list contains", list.contains?(2), true)
check("list not contains", list.contains?(9), false)

# ----- number theory: gcd / lcm -----

def gcd(a, b)
  while b != 0
    t = b
    b = a % b
    a = t
  end
  a
end

def lcm(a, b)
  a / gcd(a, b) * b
end

check("gcd", gcd(48, 36), 12)
check("gcd coprime", gcd(17, 5), 1)
check("gcd one", gcd(7, 0), 7)
check("lcm", lcm(4, 6), 12)
check("lcm coprime", lcm(3, 5), 15)

# gcd of a whole array via reduce-by-hand
def gcd_all(arr)
  g = arr[0]
  i = 1
  while i < arr.size
    g = gcd(g, arr[i])
    i += 1
  end
  g
end
check("gcd all", gcd_all([24, 36, 48, 60]), 12)

# ----- frequency table (hash as a multiset) -----

def frequencies(arr)
  freq = {}
  arr.each do |x|
    if freq.include?(x)
      freq[x] = freq[x] + 1
    else
      freq[x] = 1
    end
  end
  freq
end

f = frequencies([1, 2, 2, 3, 3, 3, 1])
check("freq of 1", f[1], 2)
check("freq of 2", f[2], 2)
check("freq of 3", f[3], 3)
check("freq distinct keys", f.keys.size, 3)
check("freq total", f.values.sum, 7)

# the most frequent element (mode)
def mode(arr)
  freq = frequencies(arr)
  best = nil
  best_count = 0
  freq.keys.each do |k|
    if freq[k] > best_count
      best_count = freq[k]
      best = k
    end
  end
  best
end
check("mode", mode([4, 4, 5, 4, 6, 5]), 4)

# ----- min / max / sum without built-in reducers -----

def minimum(arr)
  m = arr[0]
  arr.each { |x| m = x if x < m }
  m
end

def maximum(arr)
  m = arr[0]
  arr.each { |x| m = x if x > m }
  m
end

check("minimum", minimum([7, 3, 9, 1, 5]), 1)
check("maximum", maximum([7, 3, 9, 1, 5]), 9)
check("sum builtin", [7, 3, 9, 1, 5].sum, 25)
check("range from min to max", maximum(data) - minimum(data), 9)

# ----- done -----
if fails == 0
  puts "Ruby big self test 1 (data structures) passed"
end
exit(fails)

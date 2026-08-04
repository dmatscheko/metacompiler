@AR = global [2097152 x i8] zeroinitializer
@APOS = global i32 zeroinitializer
@EMPTY = global [1 x i8] zeroinitializer
@args = global [2560 x i32*] zeroinitializer
@frame = global i32 zeroinitializer
@CAP_BUFS = global [8 x i32*] zeroinitializer
@CAP_LENS = global [8 x i32] zeroinitializer
@CAP_SP = global i32 zeroinitializer
@.str.1 = global [4 x i8] zeroinitializer
@.str.2 = global [4 x i8] zeroinitializer
@.str.3 = global [4 x i8] zeroinitializer
@.str.4 = global [4 x i8] zeroinitializer
@.str.5 = global [5 x i8] zeroinitializer
@.str.6 = global [5 x i8] zeroinitializer
@.str.7 = global [5 x i8] zeroinitializer
@.str.8 = global [5 x i8] zeroinitializer
@.str.9 = global [5 x i8] zeroinitializer
@.str.10 = global [5 x i8] zeroinitializer
@.str.11 = global [5 x i8] zeroinitializer
@.str.12 = global [5 x i8] zeroinitializer
@.str.13 = global [5 x i8] zeroinitializer
@.str.14 = global [5 x i8] zeroinitializer
@.str.15 = global [5 x i8] zeroinitializer
@.str.16 = global [5 x i8] zeroinitializer
@.str.17 = global [5 x i8] zeroinitializer
@.str.18 = global [5 x i8] zeroinitializer
@.str.19 = global [5 x i8] zeroinitializer
@.str.20 = global [5 x i8] zeroinitializer
@.str.21 = global [5 x i8] zeroinitializer
@.str.22 = global [5 x i8] zeroinitializer
@.str.23 = global [3 x i8] zeroinitializer
@.str.24 = global [4 x i8] zeroinitializer
@.str.25 = global [11 x i8] zeroinitializer
@.str.26 = global [12 x i8] zeroinitializer
@.str.27 = global [20 x i8] zeroinitializer
@.str.28 = global [21 x i8] zeroinitializer
@.str.29 = global [9 x i8] zeroinitializer
@.str.30 = global [10 x i8] zeroinitializer
@.str.31 = global [17 x i8] zeroinitializer
@.str.32 = global [18 x i8] zeroinitializer
@.str.33 = global [15 x i8] zeroinitializer
@.str.34 = global [16 x i8] zeroinitializer

define i32* @rt_bump(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32*
	%3 = load i32, i32* @APOS
	%4 = getelementptr [2097152 x i8], [2097152 x i8]* @AR, i32 0, i32 %3
	%5 = bitcast i8* %4 to i32*
	store i32* %5, i32** %2
	%6 = load i32, i32* @APOS
	%7 = load i32, i32* %1
	%8 = add i32 %6, %7
	store i32 %8, i32* @APOS
	%9 = load i32*, i32** %2
	%10 = bitcast i32* %9 to i32*
	ret i32* %10

dead1:
	ret i32* null
}

define i32 @rt_strlen(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	br label %3

3:
	%4 = load i32, i32* %2
	%5 = load i32*, i32** %1
	%6 = getelementptr i8, i32* %5, i32 %4
	%7 = load i8, i8* %6
	%8 = sext i8 %7 to i32
	%9 = icmp ne i32 %8, 0
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %15

12:
	%13 = load i32, i32* %2
	%14 = add i32 %13, 1
	store i32 %14, i32* %2
	br label %3

15:
	%16 = load i32, i32* %2
	ret i32 %16

dead2:
	ret i32 0
}

define i32 @rt_streq(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = icmp ne i32 1, 0
	br i1 %6, label %7, label %37

7:
	%8 = alloca i8
	%9 = load i32, i32* %4
	%10 = load i32*, i32** %2
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = load i8, i8* %11
	%13 = sext i8 %12 to i32
	%14 = shl i32 %13, 24
	%15 = ashr i32 %14, 24
	%16 = shl i32 %15, 24
	%17 = ashr i32 %16, 24
	%18 = trunc i32 %17 to i8
	store i8 %18, i8* %8
	%19 = alloca i8
	%20 = load i32, i32* %4
	%21 = load i32*, i32** %3
	%22 = getelementptr i8, i32* %21, i32 %20
	%23 = load i8, i8* %22
	%24 = sext i8 %23 to i32
	%25 = shl i32 %24, 24
	%26 = ashr i32 %25, 24
	%27 = shl i32 %26, 24
	%28 = ashr i32 %27, 24
	%29 = trunc i32 %28 to i8
	store i8 %29, i8* %19
	%30 = load i8, i8* %8
	%31 = sext i8 %30 to i32
	%32 = load i8, i8* %19
	%33 = sext i8 %32 to i32
	%34 = icmp ne i32 %31, %33
	%35 = zext i1 %34 to i32
	%36 = icmp ne i32 %35, 0
	br i1 %36, label %38, label %39

37:
	ret i32 0

38:
	ret i32 0

39:
	%40 = load i8, i8* %8
	%41 = sext i8 %40 to i32
	%42 = icmp eq i32 %41, 0
	%43 = zext i1 %42 to i32
	%44 = icmp ne i32 %43, 0
	br i1 %44, label %45, label %46

dead3:
	br label %39

45:
	ret i32 1

46:
	%47 = load i32, i32* %4
	%48 = add i32 %47, 1
	store i32 %48, i32* %4
	br label %5

dead4:
	br label %46

dead5:
	ret i32 0
}

define i32 @rt_lc(i32 %0) {
entry:
	%1 = alloca i8
	%2 = trunc i32 %0 to i8
	store i8 %2, i8* %1
	%3 = load i8, i8* %1
	%4 = sext i8 %3 to i32
	%5 = icmp sge i32 %4, 65
	%6 = zext i1 %5 to i32
	%7 = icmp ne i32 %6, 0
	%8 = zext i1 %7 to i32
	%9 = icmp ne i32 %8, 0
	br i1 %9, label %10, label %17

10:
	%11 = load i8, i8* %1
	%12 = sext i8 %11 to i32
	%13 = icmp sle i32 %12, 90
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	%16 = zext i1 %15 to i32
	br label %17

17:
	%18 = phi i32 [ %8, %entry ], [ %16, %10 ]
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %24

20:
	%21 = load i8, i8* %1
	%22 = sext i8 %21 to i32
	%23 = add i32 %22, 32
	ret i32 %23

24:
	%25 = load i8, i8* %1
	%26 = sext i8 %25 to i32
	ret i32 %26

dead6:
	br label %24

dead7:
	ret i32 0
}

define i32 @rt_streqi(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = icmp ne i32 1, 0
	br i1 %6, label %7, label %39

7:
	%8 = alloca i8
	%9 = load i32, i32* %4
	%10 = load i32*, i32** %2
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = load i8, i8* %11
	%13 = sext i8 %12 to i32
	%14 = call i32 @rt_lc(i32 %13)
	%15 = shl i32 %14, 24
	%16 = ashr i32 %15, 24
	%17 = shl i32 %16, 24
	%18 = ashr i32 %17, 24
	%19 = trunc i32 %18 to i8
	store i8 %19, i8* %8
	%20 = alloca i8
	%21 = load i32, i32* %4
	%22 = load i32*, i32** %3
	%23 = getelementptr i8, i32* %22, i32 %21
	%24 = load i8, i8* %23
	%25 = sext i8 %24 to i32
	%26 = call i32 @rt_lc(i32 %25)
	%27 = shl i32 %26, 24
	%28 = ashr i32 %27, 24
	%29 = shl i32 %28, 24
	%30 = ashr i32 %29, 24
	%31 = trunc i32 %30 to i8
	store i8 %31, i8* %20
	%32 = load i8, i8* %8
	%33 = sext i8 %32 to i32
	%34 = load i8, i8* %20
	%35 = sext i8 %34 to i32
	%36 = icmp ne i32 %33, %35
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %40, label %41

39:
	ret i32 0

40:
	ret i32 0

41:
	%42 = load i8, i8* %8
	%43 = sext i8 %42 to i32
	%44 = icmp eq i32 %43, 0
	%45 = zext i1 %44 to i32
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %48

dead8:
	br label %41

47:
	ret i32 1

48:
	%49 = load i32, i32* %4
	%50 = add i32 %49, 1
	store i32 %50, i32* %4
	br label %5

dead9:
	br label %48

dead10:
	ret i32 0
}

define i32* @rt_strcat(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = call i32 @rt_strlen(i32* %6)
	store i32 %7, i32* %4
	%8 = alloca i32
	%9 = load i32*, i32** %3
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_strlen(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32*
	%13 = load i32, i32* %4
	%14 = load i32, i32* %8
	%15 = add i32 %13, %14
	%16 = add i32 %15, 1
	%17 = call i32* @rt_bump(i32 %16)
	%18 = bitcast i32* %17 to i32*
	store i32* %18, i32** %12
	%19 = alloca i32
	store i32 0, i32* %19
	%20 = alloca i32
	store i32 0, i32* %20
	br label %21

21:
	%22 = load i32, i32* %20
	%23 = load i32*, i32** %2
	%24 = getelementptr i8, i32* %23, i32 %22
	%25 = load i8, i8* %24
	%26 = sext i8 %25 to i32
	%27 = icmp ne i32 %26, 0
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %30, label %48

30:
	%31 = load i32, i32* %19
	%32 = load i32*, i32** %12
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = load i32, i32* %20
	%35 = load i32*, i32** %2
	%36 = getelementptr i8, i32* %35, i32 %34
	%37 = load i8, i8* %36
	%38 = sext i8 %37 to i32
	%39 = shl i32 %38, 24
	%40 = ashr i32 %39, 24
	%41 = shl i32 %40, 24
	%42 = ashr i32 %41, 24
	%43 = trunc i32 %42 to i8
	store i8 %43, i8* %33
	%44 = load i32, i32* %19
	%45 = add i32 %44, 1
	store i32 %45, i32* %19
	%46 = load i32, i32* %20
	%47 = add i32 %46, 1
	store i32 %47, i32* %20
	br label %21

48:
	store i32 0, i32* %20
	br label %49

49:
	%50 = load i32, i32* %20
	%51 = load i32*, i32** %3
	%52 = getelementptr i8, i32* %51, i32 %50
	%53 = load i8, i8* %52
	%54 = sext i8 %53 to i32
	%55 = icmp ne i32 %54, 0
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %58, label %76

58:
	%59 = load i32, i32* %19
	%60 = load i32*, i32** %12
	%61 = getelementptr i8, i32* %60, i32 %59
	%62 = load i32, i32* %20
	%63 = load i32*, i32** %3
	%64 = getelementptr i8, i32* %63, i32 %62
	%65 = load i8, i8* %64
	%66 = sext i8 %65 to i32
	%67 = shl i32 %66, 24
	%68 = ashr i32 %67, 24
	%69 = shl i32 %68, 24
	%70 = ashr i32 %69, 24
	%71 = trunc i32 %70 to i8
	store i8 %71, i8* %61
	%72 = load i32, i32* %19
	%73 = add i32 %72, 1
	store i32 %73, i32* %19
	%74 = load i32, i32* %20
	%75 = add i32 %74, 1
	store i32 %75, i32* %20
	br label %49

76:
	%77 = load i32, i32* %19
	%78 = load i32*, i32** %12
	%79 = getelementptr i8, i32* %78, i32 %77
	%80 = shl i32 0, 24
	%81 = ashr i32 %80, 24
	%82 = shl i32 %81, 24
	%83 = ashr i32 %82, 24
	%84 = trunc i32 %83 to i8
	store i8 %84, i8* %79
	%85 = load i32*, i32** %12
	%86 = bitcast i32* %85 to i32*
	ret i32* %86

dead11:
	ret i32* null
}

define i32* @rt_sub(i32* %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = load i32, i32* %5
	%8 = load i32, i32* %4
	%9 = sub i32 %7, %8
	store i32 %9, i32* %6
	%10 = load i32, i32* %6
	%11 = icmp slt i32 %10, 0
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	br i1 %13, label %14, label %15

14:
	store i32 0, i32* %6
	br label %15

15:
	%16 = alloca i32*
	%17 = load i32, i32* %6
	%18 = add i32 %17, 1
	%19 = call i32* @rt_bump(i32 %18)
	%20 = bitcast i32* %19 to i32*
	store i32* %20, i32** %16
	%21 = alloca i32
	store i32 0, i32* %21
	br label %22

22:
	%23 = load i32, i32* %21
	%24 = load i32, i32* %6
	%25 = icmp slt i32 %23, %24
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	br i1 %27, label %28, label %46

28:
	%29 = load i32, i32* %21
	%30 = load i32*, i32** %16
	%31 = getelementptr i8, i32* %30, i32 %29
	%32 = load i32, i32* %4
	%33 = load i32, i32* %21
	%34 = add i32 %32, %33
	%35 = load i32*, i32** %3
	%36 = getelementptr i8, i32* %35, i32 %34
	%37 = load i8, i8* %36
	%38 = sext i8 %37 to i32
	%39 = shl i32 %38, 24
	%40 = ashr i32 %39, 24
	%41 = shl i32 %40, 24
	%42 = ashr i32 %41, 24
	%43 = trunc i32 %42 to i8
	store i8 %43, i8* %31
	%44 = load i32, i32* %21
	%45 = add i32 %44, 1
	store i32 %45, i32* %21
	br label %22

46:
	%47 = load i32, i32* %6
	%48 = load i32*, i32** %16
	%49 = getelementptr i8, i32* %48, i32 %47
	%50 = shl i32 0, 24
	%51 = ashr i32 %50, 24
	%52 = shl i32 %51, 24
	%53 = ashr i32 %52, 24
	%54 = trunc i32 %53 to i8
	store i8 %54, i8* %49
	%55 = load i32*, i32** %16
	%56 = bitcast i32* %55 to i32*
	ret i32* %56

dead12:
	ret i32* null
}

define i32 @rt_lastch(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	store i32 0, i32* %4
	%5 = alloca i32
	store i32 -1, i32* %5
	br label %6

6:
	%7 = icmp ne i32 1, 0
	br i1 %7, label %8, label %19

8:
	%9 = alloca i32
	%10 = load i32, i32* %4
	%11 = load i32*, i32** %2
	%12 = getelementptr i8, i32* %11, i32 %10
	%13 = load i8, i8* %12
	%14 = sext i8 %13 to i32
	store i32 %14, i32* %9
	%15 = load i32, i32* %9
	%16 = icmp eq i32 %15, 0
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %21, label %23

19:
	%20 = load i32, i32* %5
	ret i32 %20

21:
	%22 = load i32, i32* %5
	ret i32 %22

23:
	%24 = load i32, i32* %9
	%25 = load i32, i32* %3
	%26 = icmp eq i32 %24, %25
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %31

dead13:
	br label %23

29:
	%30 = load i32, i32* %4
	store i32 %30, i32* %5
	br label %31

31:
	%32 = load i32, i32* %4
	%33 = add i32 %32, 1
	store i32 %33, i32* %4
	br label %6

dead14:
	ret i32 0
}

define i32 @rt_findch(i32* %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = load i32, i32* %5
	store i32 %7, i32* %6
	br label %8

8:
	%9 = icmp ne i32 1, 0
	br i1 %9, label %10, label %21

10:
	%11 = alloca i32
	%12 = load i32, i32* %6
	%13 = load i32*, i32** %3
	%14 = getelementptr i8, i32* %13, i32 %12
	%15 = load i8, i8* %14
	%16 = sext i8 %15 to i32
	store i32 %16, i32* %11
	%17 = load i32, i32* %11
	%18 = icmp eq i32 %17, 0
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %22, label %23

21:
	ret i32 -1

22:
	ret i32 -1

23:
	%24 = load i32, i32* %11
	%25 = load i32, i32* %4
	%26 = icmp eq i32 %24, %25
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %31

dead15:
	br label %23

29:
	%30 = load i32, i32* %6
	ret i32 %30

31:
	%32 = load i32, i32* %6
	%33 = add i32 %32, 1
	store i32 %33, i32* %6
	br label %8

dead16:
	br label %31

dead17:
	ret i32 0
}

define i32 @rt_findstr(i32* %0, i32* %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32*
	store i32* %1, i32** %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_strlen(i32* %8)
	store i32 %9, i32* %6
	%10 = alloca i32
	%11 = load i32*, i32** %4
	%12 = bitcast i32* %11 to i32*
	%13 = call i32 @rt_strlen(i32* %12)
	store i32 %13, i32* %10
	%14 = alloca i32
	%15 = load i32, i32* %5
	store i32 %15, i32* %14
	br label %16

16:
	%17 = load i32, i32* %14
	%18 = load i32, i32* %10
	%19 = add i32 %17, %18
	%20 = load i32, i32* %6
	%21 = icmp sle i32 %19, %20
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	br i1 %23, label %24, label %27

24:
	%25 = alloca i32
	store i32 0, i32* %25
	%26 = alloca i32
	store i32 0, i32* %26
	br label %28

27:
	ret i32 -1

28:
	%29 = load i32, i32* %26
	%30 = icmp eq i32 %29, 0
	%31 = zext i1 %30 to i32
	%32 = icmp ne i32 %31, 0
	%33 = zext i1 %32 to i32
	%34 = icmp ne i32 %33, 0
	br i1 %34, label %58, label %65

35:
	%36 = load i32, i32* %14
	%37 = load i32, i32* %25
	%38 = add i32 %36, %37
	%39 = load i32*, i32** %3
	%40 = getelementptr i8, i32* %39, i32 %38
	%41 = load i8, i8* %40
	%42 = sext i8 %41 to i32
	%43 = call i32 @rt_lc(i32 %42)
	%44 = load i32, i32* %25
	%45 = load i32*, i32** %4
	%46 = getelementptr i8, i32* %45, i32 %44
	%47 = load i8, i8* %46
	%48 = sext i8 %47 to i32
	%49 = call i32 @rt_lc(i32 %48)
	%50 = icmp ne i32 %43, %49
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %68, label %70

53:
	%54 = load i32, i32* %26
	%55 = icmp eq i32 %54, 0
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %73, label %75

58:
	%59 = load i32, i32* %25
	%60 = load i32, i32* %10
	%61 = icmp slt i32 %59, %60
	%62 = zext i1 %61 to i32
	%63 = icmp ne i32 %62, 0
	%64 = zext i1 %63 to i32
	br label %65

65:
	%66 = phi i32 [ %33, %28 ], [ %64, %58 ]
	%67 = icmp ne i32 %66, 0
	br i1 %67, label %35, label %53

68:
	store i32 1, i32* %26
	br label %69

69:
	br label %28

70:
	%71 = load i32, i32* %25
	%72 = add i32 %71, 1
	store i32 %72, i32* %25
	br label %69

73:
	%74 = load i32, i32* %14
	ret i32 %74

75:
	%76 = load i32, i32* %14
	%77 = add i32 %76, 1
	store i32 %77, i32* %14
	br label %16

dead18:
	br label %75

dead19:
	ret i32 0
}

define i32* @rt_int2str(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca [16 x i8]
	%3 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 0
	store i8 0, i8* %3
	%4 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 1
	store i8 0, i8* %4
	%5 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 2
	store i8 0, i8* %5
	%6 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 3
	store i8 0, i8* %6
	%7 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 4
	store i8 0, i8* %7
	%8 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 5
	store i8 0, i8* %8
	%9 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 6
	store i8 0, i8* %9
	%10 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 7
	store i8 0, i8* %10
	%11 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 8
	store i8 0, i8* %11
	%12 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 9
	store i8 0, i8* %12
	%13 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 10
	store i8 0, i8* %13
	%14 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 11
	store i8 0, i8* %14
	%15 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 12
	store i8 0, i8* %15
	%16 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 13
	store i8 0, i8* %16
	%17 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 14
	store i8 0, i8* %17
	%18 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 15
	store i8 0, i8* %18
	%19 = alloca i32*
	%20 = call i32* @rt_bump(i32 16)
	%21 = bitcast i32* %20 to i32*
	store i32* %21, i32** %19
	%22 = load i32, i32* %1
	%23 = icmp eq i32 %22, 0
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %43

26:
	%27 = load i32*, i32** %19
	%28 = getelementptr i8, i32* %27, i32 0
	%29 = shl i32 48, 24
	%30 = ashr i32 %29, 24
	%31 = shl i32 %30, 24
	%32 = ashr i32 %31, 24
	%33 = trunc i32 %32 to i8
	store i8 %33, i8* %28
	%34 = load i32*, i32** %19
	%35 = getelementptr i8, i32* %34, i32 1
	%36 = shl i32 0, 24
	%37 = ashr i32 %36, 24
	%38 = shl i32 %37, 24
	%39 = ashr i32 %38, 24
	%40 = trunc i32 %39 to i8
	store i8 %40, i8* %35
	%41 = load i32*, i32** %19
	%42 = bitcast i32* %41 to i32*
	ret i32* %42

43:
	%44 = alloca i32
	%45 = load i32, i32* %1
	%46 = icmp slt i32 %45, 0
	%47 = zext i1 %46 to i32
	store i32 %47, i32* %44
	%48 = alloca i32
	%49 = load i32, i32* %44
	%50 = icmp ne i32 %49, 0
	br i1 %50, label %51, label %54

dead20:
	br label %43

51:
	%52 = load i32, i32* %1
	%53 = sub i32 0, %52
	br label %56

54:
	%55 = load i32, i32* %1
	br label %56

56:
	%57 = phi i32 [ %53, %51 ], [ %55, %54 ]
	store i32 %57, i32* %48
	%58 = alloca i32
	store i32 15, i32* %58
	br label %59

59:
	%60 = load i32, i32* %48
	%61 = icmp sgt i32 %60, 0
	%62 = zext i1 %61 to i32
	%63 = icmp ne i32 %62, 0
	br i1 %63, label %64, label %79

64:
	%65 = load i32, i32* %58
	%66 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 %65
	%67 = load i32, i32* %48
	%68 = srem i32 %67, 10
	%69 = add i32 48, %68
	%70 = shl i32 %69, 24
	%71 = ashr i32 %70, 24
	%72 = shl i32 %71, 24
	%73 = ashr i32 %72, 24
	%74 = trunc i32 %73 to i8
	store i8 %74, i8* %66
	%75 = load i32, i32* %58
	%76 = sub i32 %75, 1
	store i32 %76, i32* %58
	%77 = load i32, i32* %48
	%78 = sdiv i32 %77, 10
	store i32 %78, i32* %48
	br label %59

79:
	%80 = alloca i32
	store i32 0, i32* %80
	%81 = load i32, i32* %44
	%82 = icmp ne i32 %81, 0
	br i1 %82, label %83, label %91

83:
	%84 = load i32*, i32** %19
	%85 = getelementptr i8, i32* %84, i32 0
	%86 = shl i32 45, 24
	%87 = ashr i32 %86, 24
	%88 = shl i32 %87, 24
	%89 = ashr i32 %88, 24
	%90 = trunc i32 %89 to i8
	store i8 %90, i8* %85
	store i32 1, i32* %80
	br label %91

91:
	%92 = alloca i32
	%93 = load i32, i32* %58
	%94 = add i32 %93, 1
	store i32 %94, i32* %92
	br label %95

95:
	%96 = load i32, i32* %92
	%97 = icmp sle i32 %96, 15
	%98 = zext i1 %97 to i32
	%99 = icmp ne i32 %98, 0
	br i1 %99, label %100, label %117

100:
	%101 = load i32, i32* %80
	%102 = load i32*, i32** %19
	%103 = getelementptr i8, i32* %102, i32 %101
	%104 = load i32, i32* %92
	%105 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 %104
	%106 = load i8, i8* %105
	%107 = sext i8 %106 to i32
	%108 = shl i32 %107, 24
	%109 = ashr i32 %108, 24
	%110 = shl i32 %109, 24
	%111 = ashr i32 %110, 24
	%112 = trunc i32 %111 to i8
	store i8 %112, i8* %103
	%113 = load i32, i32* %80
	%114 = add i32 %113, 1
	store i32 %114, i32* %80
	%115 = load i32, i32* %92
	%116 = add i32 %115, 1
	store i32 %116, i32* %92
	br label %95

117:
	%118 = load i32, i32* %80
	%119 = load i32*, i32** %19
	%120 = getelementptr i8, i32* %119, i32 %118
	%121 = shl i32 0, 24
	%122 = ashr i32 %121, 24
	%123 = shl i32 %122, 24
	%124 = ashr i32 %123, 24
	%125 = trunc i32 %124 to i8
	store i8 %125, i8* %120
	%126 = load i32*, i32** %19
	%127 = bitcast i32* %126 to i32*
	ret i32* %127

dead21:
	ret i32* null
}

define i32 @rt_str2int(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = alloca i32
	store i32 0, i32* %3
	%4 = alloca i32
	store i32 1, i32* %4
	br label %5

5:
	%6 = load i32, i32* %2
	%7 = load i32*, i32** %1
	%8 = getelementptr i8, i32* %7, i32 %6
	%9 = load i8, i8* %8
	%10 = sext i8 %9 to i32
	%11 = icmp eq i32 %10, 32
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %40, label %30

16:
	%17 = load i32, i32* %2
	%18 = add i32 %17, 1
	store i32 %18, i32* %2
	br label %5

19:
	%20 = load i32, i32* %2
	%21 = load i32*, i32** %1
	%22 = getelementptr i8, i32* %21, i32 %20
	%23 = load i8, i8* %22
	%24 = sext i8 %23 to i32
	%25 = icmp eq i32 %24, 45
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %53, label %43

30:
	%31 = load i32, i32* %2
	%32 = load i32*, i32** %1
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = load i8, i8* %33
	%35 = sext i8 %34 to i32
	%36 = icmp eq i32 %35, 9
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	%39 = zext i1 %38 to i32
	br label %40

40:
	%41 = phi i32 [ %14, %5 ], [ %39, %30 ]
	%42 = icmp ne i32 %41, 0
	br i1 %42, label %16, label %19

43:
	%44 = load i32, i32* %2
	%45 = load i32*, i32** %1
	%46 = getelementptr i8, i32* %45, i32 %44
	%47 = load i8, i8* %46
	%48 = sext i8 %47 to i32
	%49 = icmp eq i32 %48, 43
	%50 = zext i1 %49 to i32
	%51 = icmp ne i32 %50, 0
	%52 = zext i1 %51 to i32
	br label %53

53:
	%54 = phi i32 [ %28, %19 ], [ %52, %43 ]
	%55 = icmp ne i32 %54, 0
	br i1 %55, label %56, label %65

56:
	%57 = load i32, i32* %2
	%58 = load i32*, i32** %1
	%59 = getelementptr i8, i32* %58, i32 %57
	%60 = load i8, i8* %59
	%61 = sext i8 %60 to i32
	%62 = icmp eq i32 %61, 45
	%63 = zext i1 %62 to i32
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %66, label %67

65:
	br label %70

66:
	store i32 -1, i32* %4
	br label %67

67:
	%68 = load i32, i32* %2
	%69 = add i32 %68, 1
	store i32 %69, i32* %2
	br label %65

70:
	%71 = load i32, i32* %2
	%72 = load i32*, i32** %1
	%73 = getelementptr i8, i32* %72, i32 %71
	%74 = load i8, i8* %73
	%75 = sext i8 %74 to i32
	%76 = icmp sge i32 %75, 48
	%77 = zext i1 %76 to i32
	%78 = icmp ne i32 %77, 0
	%79 = zext i1 %78 to i32
	%80 = icmp ne i32 %79, 0
	br i1 %80, label %97, label %107

81:
	%82 = load i32, i32* %3
	%83 = mul i32 %82, 10
	%84 = load i32, i32* %2
	%85 = load i32*, i32** %1
	%86 = getelementptr i8, i32* %85, i32 %84
	%87 = load i8, i8* %86
	%88 = sext i8 %87 to i32
	%89 = sub i32 %88, 48
	%90 = add i32 %83, %89
	store i32 %90, i32* %3
	%91 = load i32, i32* %2
	%92 = add i32 %91, 1
	store i32 %92, i32* %2
	br label %70

93:
	%94 = load i32, i32* %3
	%95 = load i32, i32* %4
	%96 = mul i32 %94, %95
	ret i32 %96

97:
	%98 = load i32, i32* %2
	%99 = load i32*, i32** %1
	%100 = getelementptr i8, i32* %99, i32 %98
	%101 = load i8, i8* %100
	%102 = sext i8 %101 to i32
	%103 = icmp sle i32 %102, 57
	%104 = zext i1 %103 to i32
	%105 = icmp ne i32 %104, 0
	%106 = zext i1 %105 to i32
	br label %107

107:
	%108 = phi i32 [ %79, %70 ], [ %106, %97 ]
	%109 = icmp ne i32 %108, 0
	br i1 %109, label %81, label %93

dead22:
	ret i32 0
}

define i32 @rt_prints(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	br label %3

3:
	%4 = load i32, i32* %2
	%5 = load i32*, i32** %1
	%6 = getelementptr i8, i32* %5, i32 %4
	%7 = load i8, i8* %6
	%8 = sext i8 %7 to i32
	%9 = icmp ne i32 %8, 0
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %22

12:
	%13 = load i32, i32* %2
	%14 = load i32*, i32** %1
	%15 = getelementptr i8, i32* %14, i32 %13
	%16 = load i8, i8* %15
	%17 = sext i8 %16 to i32
	%18 = and i32 %17, 255
	%19 = call i32 @putchar(i32 %18)
	%20 = load i32, i32* %2
	%21 = add i32 %20, 1
	store i32 %21, i32* %2
	br label %3

22:
	ret i32 0

dead23:
	ret i32 0
}

declare i32 @putchar(i32 %0)

define i32 @rt_capstart() {
entry:
	%0 = load i32, i32* @CAP_SP
	%1 = getelementptr [8 x i32*], [8 x i32*]* @CAP_BUFS, i32 0, i32 %0
	%2 = call i32* @rt_bump(i32 u0x2000)
	%3 = bitcast i32* %2 to i32*
	store i32* %3, i32** %1
	%4 = load i32, i32* @CAP_SP
	%5 = getelementptr [8 x i32], [8 x i32]* @CAP_LENS, i32 0, i32 %4
	store i32 0, i32* %5
	%6 = load i32, i32* @CAP_SP
	%7 = add i32 %6, 1
	store i32 %7, i32* @CAP_SP
	ret i32 0

dead24:
	ret i32 0
}

define i32* @rt_capend() {
entry:
	%0 = load i32, i32* @CAP_SP
	%1 = sub i32 %0, 1
	store i32 %1, i32* @CAP_SP
	%2 = alloca i32*
	%3 = load i32, i32* @CAP_SP
	%4 = getelementptr [8 x i32*], [8 x i32*]* @CAP_BUFS, i32 0, i32 %3
	%5 = load i32*, i32** %4
	%6 = bitcast i32* %5 to i32*
	store i32* %6, i32** %2
	%7 = load i32, i32* @CAP_SP
	%8 = getelementptr [8 x i32], [8 x i32]* @CAP_LENS, i32 0, i32 %7
	%9 = load i32, i32* %8
	%10 = load i32*, i32** %2
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = shl i32 0, 24
	%13 = ashr i32 %12, 24
	%14 = shl i32 %13, 24
	%15 = ashr i32 %14, 24
	%16 = trunc i32 %15 to i8
	store i8 %16, i8* %11
	%17 = load i32*, i32** %2
	%18 = bitcast i32* %17 to i32*
	ret i32* %18

dead25:
	ret i32* null
}

define i32 @rt_println(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = load i32, i32* @CAP_SP
	%3 = icmp eq i32 %2, 0
	%4 = zext i1 %3 to i32
	%5 = icmp ne i32 %4, 0
	br i1 %5, label %6, label %11

6:
	%7 = load i32*, i32** %1
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_prints(i32* %8)
	%10 = call i32 @putchar(i32 10)
	ret i32 0

11:
	%12 = alloca i32
	%13 = load i32, i32* @CAP_SP
	%14 = sub i32 %13, 1
	store i32 %14, i32* %12
	%15 = alloca i32*
	%16 = load i32, i32* %12
	%17 = getelementptr [8 x i32*], [8 x i32*]* @CAP_BUFS, i32 0, i32 %16
	%18 = load i32*, i32** %17
	%19 = bitcast i32* %18 to i32*
	store i32* %19, i32** %15
	%20 = alloca i32
	%21 = load i32, i32* %12
	%22 = getelementptr [8 x i32], [8 x i32]* @CAP_LENS, i32 0, i32 %21
	%23 = load i32, i32* %22
	store i32 %23, i32* %20
	%24 = alloca i32
	store i32 0, i32* %24
	br label %25

dead26:
	br label %11

25:
	%26 = load i32, i32* %24
	%27 = load i32*, i32** %1
	%28 = getelementptr i8, i32* %27, i32 %26
	%29 = load i8, i8* %28
	%30 = sext i8 %29 to i32
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	br i1 %33, label %34, label %52

34:
	%35 = load i32, i32* %20
	%36 = load i32, i32* %24
	%37 = add i32 %35, %36
	%38 = load i32*, i32** %15
	%39 = getelementptr i8, i32* %38, i32 %37
	%40 = load i32, i32* %24
	%41 = load i32*, i32** %1
	%42 = getelementptr i8, i32* %41, i32 %40
	%43 = load i8, i8* %42
	%44 = sext i8 %43 to i32
	%45 = shl i32 %44, 24
	%46 = ashr i32 %45, 24
	%47 = shl i32 %46, 24
	%48 = ashr i32 %47, 24
	%49 = trunc i32 %48 to i8
	store i8 %49, i8* %39
	%50 = load i32, i32* %24
	%51 = add i32 %50, 1
	store i32 %51, i32* %24
	br label %25

52:
	%53 = load i32, i32* %20
	%54 = load i32, i32* %24
	%55 = add i32 %53, %54
	%56 = load i32*, i32** %15
	%57 = getelementptr i8, i32* %56, i32 %55
	%58 = shl i32 10, 24
	%59 = ashr i32 %58, 24
	%60 = shl i32 %59, 24
	%61 = ashr i32 %60, 24
	%62 = trunc i32 %61 to i8
	store i8 %62, i8* %57
	%63 = load i32, i32* %12
	%64 = getelementptr [8 x i32], [8 x i32]* @CAP_LENS, i32 0, i32 %63
	%65 = load i32, i32* %20
	%66 = load i32, i32* %24
	%67 = add i32 %65, %66
	%68 = add i32 %67, 1
	store i32 %68, i32* %64
	ret i32 0

dead27:
	ret i32 0
}

define i32* @rt_stripq(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32
	%7 = load i32, i32* %2
	%8 = icmp sge i32 %7, 2
	%9 = zext i1 %8 to i32
	store i32 %9, i32* %6
	%10 = alloca i32
	%11 = load i32, i32* %6
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %13, label %16

13:
	%14 = load i32, i32* %2
	%15 = sub i32 %14, 1
	br label %17

16:
	br label %17

17:
	%18 = phi i32 [ %15, %13 ], [ 0, %16 ]
	store i32 %18, i32* %10
	%19 = alloca i32
	%20 = load i32, i32* %6
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	br i1 %23, label %24, label %33

24:
	%25 = load i32*, i32** %1
	%26 = getelementptr i8, i32* %25, i32 0
	%27 = load i8, i8* %26
	%28 = sext i8 %27 to i32
	%29 = icmp eq i32 %28, 34
	%30 = zext i1 %29 to i32
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	br label %33

33:
	%34 = phi i32 [ %22, %17 ], [ %32, %24 ]
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %46

36:
	%37 = load i32, i32* %10
	%38 = load i32*, i32** %1
	%39 = getelementptr i8, i32* %38, i32 %37
	%40 = load i8, i8* %39
	%41 = sext i8 %40 to i32
	%42 = icmp eq i32 %41, 34
	%43 = zext i1 %42 to i32
	%44 = icmp ne i32 %43, 0
	%45 = zext i1 %44 to i32
	br label %46

46:
	%47 = phi i32 [ %34, %33 ], [ %45, %36 ]
	store i32 %47, i32* %19
	%48 = load i32*, i32** %1
	%49 = bitcast i32* %48 to i32*
	%50 = load i32, i32* %19
	%51 = icmp ne i32 %50, 0
	br i1 %51, label %52, label %53

52:
	br label %54

53:
	br label %54

54:
	%55 = phi i32 [ 1, %52 ], [ 0, %53 ]
	%56 = load i32, i32* %19
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %58, label %61

58:
	%59 = load i32, i32* %2
	%60 = sub i32 %59, 1
	br label %63

61:
	%62 = load i32, i32* %2
	br label %63

63:
	%64 = phi i32 [ %60, %58 ], [ %62, %61 ]
	%65 = call i32* @rt_sub(i32* %49, i32 %55, i32 %64)
	%66 = bitcast i32* %65 to i32*
	ret i32* %66

dead28:
	ret i32* null
}

define i32* @rt_substr(i32* %0, i32 %1, i32 %2, i32 %3) {
entry:
	%4 = alloca i32*
	store i32* %0, i32** %4
	%5 = alloca i32
	store i32 %1, i32* %5
	%6 = alloca i32
	store i32 %2, i32* %6
	%7 = alloca i32
	store i32 %3, i32* %7
	%8 = alloca i32
	%9 = load i32*, i32** %4
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_strlen(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32
	%13 = load i32, i32* %5
	%14 = icmp slt i32 %13, 0
	%15 = zext i1 %14 to i32
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %21

17:
	%18 = load i32, i32* %8
	%19 = load i32, i32* %5
	%20 = add i32 %18, %19
	br label %23

21:
	%22 = load i32, i32* %5
	br label %23

23:
	%24 = phi i32 [ %20, %17 ], [ %22, %21 ]
	store i32 %24, i32* %12
	%25 = load i32, i32* %12
	%26 = icmp slt i32 %25, 0
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %30

29:
	store i32 0, i32* %12
	br label %30

30:
	%31 = load i32, i32* %12
	%32 = load i32, i32* %8
	%33 = icmp sgt i32 %31, %32
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %38

36:
	%37 = load i32, i32* %8
	store i32 %37, i32* %12
	br label %38

38:
	%39 = alloca i32
	%40 = load i32, i32* %7
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	%43 = icmp ne i32 %42, 0
	br i1 %43, label %44, label %49

44:
	%45 = load i32, i32* %6
	%46 = icmp slt i32 %45, 0
	%47 = zext i1 %46 to i32
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %58, label %62

49:
	%50 = load i32, i32* %8
	br label %51

51:
	%52 = phi i32 [ %67, %66 ], [ %50, %49 ]
	store i32 %52, i32* %39
	%53 = load i32, i32* %39
	%54 = load i32, i32* %12
	%55 = icmp slt i32 %53, %54
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %68, label %70

58:
	%59 = load i32, i32* %8
	%60 = load i32, i32* %6
	%61 = add i32 %59, %60
	br label %66

62:
	%63 = load i32, i32* %12
	%64 = load i32, i32* %6
	%65 = add i32 %63, %64
	br label %66

66:
	%67 = phi i32 [ %61, %58 ], [ %65, %62 ]
	br label %51

68:
	%69 = load i32, i32* %12
	store i32 %69, i32* %39
	br label %70

70:
	%71 = load i32, i32* %39
	%72 = load i32, i32* %8
	%73 = icmp sgt i32 %71, %72
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	br i1 %75, label %76, label %78

76:
	%77 = load i32, i32* %8
	store i32 %77, i32* %39
	br label %78

78:
	%79 = load i32*, i32** %4
	%80 = bitcast i32* %79 to i32*
	%81 = load i32, i32* %12
	%82 = load i32, i32* %39
	%83 = call i32* @rt_sub(i32* %80, i32 %81, i32 %82)
	%84 = bitcast i32* %83 to i32*
	ret i32* %84

dead29:
	ret i32* null
}

define i32* @rt_subst(i32* %0, i32* %1, i32* %2, i32 %3) {
entry:
	%4 = alloca i32*
	store i32* %0, i32** %4
	%5 = alloca i32*
	store i32* %1, i32** %5
	%6 = alloca i32*
	store i32* %2, i32** %6
	%7 = alloca i32
	store i32 %3, i32* %7
	%8 = alloca i32
	%9 = load i32*, i32** %5
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_strlen(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32
	%13 = load i32*, i32** %4
	%14 = bitcast i32* %13 to i32*
	%15 = call i32 @rt_strlen(i32* %14)
	store i32 %15, i32* %12
	%16 = load i32, i32* %8
	%17 = icmp eq i32 %16, 0
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %23

20:
	%21 = load i32*, i32** %4
	%22 = bitcast i32* %21 to i32*
	ret i32* %22

23:
	%24 = load i32, i32* %7
	%25 = icmp ne i32 %24, 0
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	br i1 %27, label %28, label %39

dead30:
	br label %23

28:
	%29 = alloca i32
	%30 = load i32*, i32** %4
	%31 = bitcast i32* %30 to i32*
	%32 = load i32*, i32** %5
	%33 = bitcast i32* %32 to i32*
	%34 = call i32 @rt_findstr(i32* %31, i32* %33, i32 0)
	store i32 %34, i32* %29
	%35 = load i32, i32* %29
	%36 = icmp slt i32 %35, 0
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %44, label %47

39:
	%40 = alloca i32*
	%41 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%42 = bitcast i8* %41 to i32*
	store i32* %42, i32** %40
	%43 = alloca i32
	store i32 0, i32* %43
	br label %60

44:
	%45 = load i32*, i32** %4
	%46 = bitcast i32* %45 to i32*
	ret i32* %46

47:
	%48 = load i32*, i32** %6
	%49 = bitcast i32* %48 to i32*
	%50 = load i32*, i32** %4
	%51 = bitcast i32* %50 to i32*
	%52 = load i32, i32* %29
	%53 = load i32, i32* %8
	%54 = add i32 %52, %53
	%55 = load i32, i32* %12
	%56 = call i32* @rt_sub(i32* %51, i32 %54, i32 %55)
	%57 = bitcast i32* %56 to i32*
	%58 = call i32* @rt_strcat(i32* %49, i32* %57)
	%59 = bitcast i32* %58 to i32*
	ret i32* %59

dead31:
	br label %47

dead32:
	br label %39

60:
	%61 = icmp ne i32 1, 0
	br i1 %61, label %62, label %74

62:
	%63 = alloca i32
	%64 = load i32*, i32** %4
	%65 = bitcast i32* %64 to i32*
	%66 = load i32*, i32** %5
	%67 = bitcast i32* %66 to i32*
	%68 = load i32, i32* %43
	%69 = call i32 @rt_findstr(i32* %65, i32* %67, i32 %68)
	store i32 %69, i32* %63
	%70 = load i32, i32* %63
	%71 = icmp slt i32 %70, 0
	%72 = zext i1 %71 to i32
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %77, label %88

74:
	%75 = load i32*, i32** %40
	%76 = bitcast i32* %75 to i32*
	ret i32* %76

77:
	%78 = load i32*, i32** %40
	%79 = bitcast i32* %78 to i32*
	%80 = load i32*, i32** %4
	%81 = bitcast i32* %80 to i32*
	%82 = load i32, i32* %43
	%83 = load i32, i32* %12
	%84 = call i32* @rt_sub(i32* %81, i32 %82, i32 %83)
	%85 = bitcast i32* %84 to i32*
	%86 = call i32* @rt_strcat(i32* %79, i32* %85)
	%87 = bitcast i32* %86 to i32*
	ret i32* %87

88:
	%89 = load i32*, i32** %40
	%90 = bitcast i32* %89 to i32*
	%91 = load i32*, i32** %4
	%92 = bitcast i32* %91 to i32*
	%93 = load i32, i32* %43
	%94 = load i32, i32* %63
	%95 = call i32* @rt_sub(i32* %92, i32 %93, i32 %94)
	%96 = bitcast i32* %95 to i32*
	%97 = call i32* @rt_strcat(i32* %90, i32* %96)
	%98 = bitcast i32* %97 to i32*
	%99 = load i32*, i32** %6
	%100 = bitcast i32* %99 to i32*
	%101 = call i32* @rt_strcat(i32* %98, i32* %100)
	%102 = bitcast i32* %101 to i32*
	store i32* %102, i32** %40
	%103 = load i32, i32* %63
	%104 = load i32, i32* %8
	%105 = add i32 %103, %104
	store i32 %105, i32* %43
	br label %60

dead33:
	br label %88

dead34:
	ret i32* null
}

define i32* @rt_mods(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32*
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = call i32* @rt_stripq(i32* %6)
	%8 = bitcast i32* %7 to i32*
	store i32* %8, i32** %4
	%9 = alloca i32
	%10 = load i32*, i32** %4
	%11 = bitcast i32* %10 to i32*
	%12 = call i32 @rt_strlen(i32* %11)
	store i32 %12, i32* %9
	%13 = alloca i32
	%14 = load i32, i32* %9
	%15 = icmp sge i32 %14, 2
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %18, label %19

18:
	br label %20

19:
	br label %20

20:
	%21 = phi i32 [ 1, %18 ], [ 0, %19 ]
	store i32 %21, i32* %13
	%22 = alloca i32
	%23 = load i32, i32* %9
	%24 = icmp sge i32 %23, 2
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %39

29:
	%30 = load i32, i32* %13
	%31 = load i32*, i32** %4
	%32 = getelementptr i8, i32* %31, i32 %30
	%33 = load i8, i8* %32
	%34 = sext i8 %33 to i32
	%35 = icmp eq i32 %34, 58
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	%38 = zext i1 %37 to i32
	br label %39

39:
	%40 = phi i32 [ %27, %20 ], [ %38, %29 ]
	store i32 %40, i32* %22
	%41 = alloca i32
	%42 = load i32, i32* %22
	%43 = icmp ne i32 %42, 0
	br i1 %43, label %44, label %45

44:
	br label %46

45:
	br label %46

46:
	%47 = phi i32 [ 2, %44 ], [ 0, %45 ]
	store i32 %47, i32* %41
	%48 = alloca i32*
	%49 = load i32*, i32** %4
	%50 = bitcast i32* %49 to i32*
	%51 = load i32, i32* %41
	%52 = call i32* @rt_sub(i32* %50, i32 0, i32 %51)
	%53 = bitcast i32* %52 to i32*
	store i32* %53, i32** %48
	%54 = alloca i32*
	%55 = load i32*, i32** %4
	%56 = bitcast i32* %55 to i32*
	%57 = load i32, i32* %41
	%58 = load i32, i32* %9
	%59 = call i32* @rt_sub(i32* %56, i32 %57, i32 %58)
	%60 = bitcast i32* %59 to i32*
	store i32* %60, i32** %54
	%61 = alloca i32
	%62 = load i32*, i32** %54
	%63 = bitcast i32* %62 to i32*
	%64 = call i32 @rt_strlen(i32* %63)
	store i32 %64, i32* %61
	%65 = alloca i32
	%66 = load i32*, i32** %54
	%67 = bitcast i32* %66 to i32*
	%68 = call i32 @rt_lastch(i32* %67, i32 92)
	store i32 %68, i32* %65
	%69 = alloca i32
	%70 = load i32*, i32** %54
	%71 = bitcast i32* %70 to i32*
	%72 = call i32 @rt_lastch(i32* %71, i32 47)
	store i32 %72, i32* %69
	%73 = alloca i32
	%74 = load i32, i32* %65
	%75 = load i32, i32* %69
	%76 = icmp sgt i32 %74, %75
	%77 = zext i1 %76 to i32
	%78 = icmp ne i32 %77, 0
	br i1 %78, label %79, label %81

79:
	%80 = load i32, i32* %65
	br label %83

81:
	%82 = load i32, i32* %69
	br label %83

83:
	%84 = phi i32 [ %80, %79 ], [ %82, %81 ]
	store i32 %84, i32* %73
	%85 = alloca i32*
	%86 = load i32*, i32** %54
	%87 = bitcast i32* %86 to i32*
	%88 = load i32, i32* %73
	%89 = add i32 %88, 1
	%90 = call i32* @rt_sub(i32* %87, i32 0, i32 %89)
	%91 = bitcast i32* %90 to i32*
	store i32* %91, i32** %85
	%92 = alloca i32*
	%93 = load i32*, i32** %54
	%94 = bitcast i32* %93 to i32*
	%95 = load i32, i32* %73
	%96 = add i32 %95, 1
	%97 = load i32, i32* %61
	%98 = call i32* @rt_sub(i32* %94, i32 %96, i32 %97)
	%99 = bitcast i32* %98 to i32*
	store i32* %99, i32** %92
	%100 = alloca i32
	%101 = load i32*, i32** %92
	%102 = bitcast i32* %101 to i32*
	%103 = call i32 @rt_strlen(i32* %102)
	store i32 %103, i32* %100
	%104 = alloca i32
	%105 = load i32*, i32** %92
	%106 = bitcast i32* %105 to i32*
	%107 = call i32 @rt_lastch(i32* %106, i32 46)
	store i32 %107, i32* %104
	%108 = alloca i32
	%109 = load i32, i32* %104
	%110 = icmp sgt i32 %109, 0
	%111 = zext i1 %110 to i32
	%112 = icmp ne i32 %111, 0
	br i1 %112, label %113, label %115

113:
	%114 = load i32, i32* %104
	br label %117

115:
	%116 = load i32, i32* %100
	br label %117

117:
	%118 = phi i32 [ %114, %113 ], [ %116, %115 ]
	store i32 %118, i32* %108
	%119 = alloca i32*
	%120 = load i32*, i32** %92
	%121 = bitcast i32* %120 to i32*
	%122 = load i32, i32* %108
	%123 = call i32* @rt_sub(i32* %121, i32 0, i32 %122)
	%124 = bitcast i32* %123 to i32*
	store i32* %124, i32** %119
	%125 = alloca i32*
	%126 = load i32*, i32** %92
	%127 = bitcast i32* %126 to i32*
	%128 = load i32, i32* %108
	%129 = load i32, i32* %100
	%130 = call i32* @rt_sub(i32* %127, i32 %128, i32 %129)
	%131 = bitcast i32* %130 to i32*
	store i32* %131, i32** %125
	%132 = alloca i32*
	%133 = load i32, i32* %3
	%134 = and i32 %133, 1
	%135 = icmp ne i32 %134, 0
	%136 = zext i1 %135 to i32
	%137 = icmp ne i32 %136, 0
	br i1 %137, label %138, label %141

138:
	%139 = load i32*, i32** %48
	%140 = bitcast i32* %139 to i32*
	br label %144

141:
	%142 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%143 = bitcast i8* %142 to i32*
	br label %144

144:
	%145 = phi i32* [ %140, %138 ], [ %143, %141 ]
	%146 = bitcast i32* %145 to i32*
	%147 = load i32, i32* %3
	%148 = and i32 %147, 2
	%149 = icmp ne i32 %148, 0
	%150 = zext i1 %149 to i32
	%151 = icmp ne i32 %150, 0
	br i1 %151, label %152, label %155

152:
	%153 = load i32*, i32** %85
	%154 = bitcast i32* %153 to i32*
	br label %158

155:
	%156 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%157 = bitcast i8* %156 to i32*
	br label %158

158:
	%159 = phi i32* [ %154, %152 ], [ %157, %155 ]
	%160 = bitcast i32* %159 to i32*
	%161 = call i32* @rt_strcat(i32* %146, i32* %160)
	%162 = bitcast i32* %161 to i32*
	store i32* %162, i32** %132
	%163 = load i32*, i32** %132
	%164 = bitcast i32* %163 to i32*
	%165 = load i32, i32* %3
	%166 = and i32 %165, 4
	%167 = icmp ne i32 %166, 0
	%168 = zext i1 %167 to i32
	%169 = icmp ne i32 %168, 0
	br i1 %169, label %170, label %173

170:
	%171 = load i32*, i32** %119
	%172 = bitcast i32* %171 to i32*
	br label %176

173:
	%174 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%175 = bitcast i8* %174 to i32*
	br label %176

176:
	%177 = phi i32* [ %172, %170 ], [ %175, %173 ]
	%178 = bitcast i32* %177 to i32*
	%179 = call i32* @rt_strcat(i32* %164, i32* %178)
	%180 = bitcast i32* %179 to i32*
	store i32* %180, i32** %132
	%181 = load i32*, i32** %132
	%182 = bitcast i32* %181 to i32*
	%183 = load i32, i32* %3
	%184 = and i32 %183, 8
	%185 = icmp ne i32 %184, 0
	%186 = zext i1 %185 to i32
	%187 = icmp ne i32 %186, 0
	br i1 %187, label %188, label %191

188:
	%189 = load i32*, i32** %125
	%190 = bitcast i32* %189 to i32*
	br label %194

191:
	%192 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%193 = bitcast i8* %192 to i32*
	br label %194

194:
	%195 = phi i32* [ %190, %188 ], [ %193, %191 ]
	%196 = bitcast i32* %195 to i32*
	%197 = call i32* @rt_strcat(i32* %182, i32* %196)
	%198 = bitcast i32* %197 to i32*
	store i32* %198, i32** %132
	%199 = load i32*, i32** %132
	%200 = bitcast i32* %199 to i32*
	ret i32* %200

dead35:
	ret i32* null
}

define i32 @rt_fskind(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32*
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32* @rt_stripq(i32* %4)
	%6 = bitcast i32* %5 to i32*
	store i32* %6, i32** %2
	%7 = load i32*, i32** %2
	%8 = bitcast i32* %7 to i32*
	%9 = getelementptr [4 x i8], [4 x i8]* @.str.1, i32 0, i32 0
	store i8 110, i8* %9
	%10 = getelementptr [4 x i8], [4 x i8]* @.str.1, i32 0, i32 1
	store i8 117, i8* %10
	%11 = getelementptr [4 x i8], [4 x i8]* @.str.1, i32 0, i32 2
	store i8 108, i8* %11
	%12 = getelementptr [4 x i8], [4 x i8]* @.str.1, i32 0, i32 3
	store i8 0, i8* %12
	%13 = getelementptr [4 x i8], [4 x i8]* @.str.1, i32 0, i32 0
	%14 = bitcast i8* %13 to i32*
	%15 = call i32 @rt_streqi(i32* %8, i32* %14)
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %18

17:
	ret i32 1

18:
	%19 = load i32*, i32** %2
	%20 = bitcast i32* %19 to i32*
	%21 = getelementptr [4 x i8], [4 x i8]* @.str.2, i32 0, i32 0
	store i8 99, i8* %21
	%22 = getelementptr [4 x i8], [4 x i8]* @.str.2, i32 0, i32 1
	store i8 111, i8* %22
	%23 = getelementptr [4 x i8], [4 x i8]* @.str.2, i32 0, i32 2
	store i8 110, i8* %23
	%24 = getelementptr [4 x i8], [4 x i8]* @.str.2, i32 0, i32 3
	store i8 0, i8* %24
	%25 = getelementptr [4 x i8], [4 x i8]* @.str.2, i32 0, i32 0
	%26 = bitcast i8* %25 to i32*
	%27 = call i32 @rt_streqi(i32* %20, i32* %26)
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %30

dead36:
	br label %18

29:
	ret i32 1

30:
	%31 = load i32*, i32** %2
	%32 = bitcast i32* %31 to i32*
	%33 = getelementptr [4 x i8], [4 x i8]* @.str.3, i32 0, i32 0
	store i8 112, i8* %33
	%34 = getelementptr [4 x i8], [4 x i8]* @.str.3, i32 0, i32 1
	store i8 114, i8* %34
	%35 = getelementptr [4 x i8], [4 x i8]* @.str.3, i32 0, i32 2
	store i8 110, i8* %35
	%36 = getelementptr [4 x i8], [4 x i8]* @.str.3, i32 0, i32 3
	store i8 0, i8* %36
	%37 = getelementptr [4 x i8], [4 x i8]* @.str.3, i32 0, i32 0
	%38 = bitcast i8* %37 to i32*
	%39 = call i32 @rt_streqi(i32* %32, i32* %38)
	%40 = icmp ne i32 %39, 0
	br i1 %40, label %41, label %42

dead37:
	br label %30

41:
	ret i32 1

42:
	%43 = load i32*, i32** %2
	%44 = bitcast i32* %43 to i32*
	%45 = getelementptr [4 x i8], [4 x i8]* @.str.4, i32 0, i32 0
	store i8 97, i8* %45
	%46 = getelementptr [4 x i8], [4 x i8]* @.str.4, i32 0, i32 1
	store i8 117, i8* %46
	%47 = getelementptr [4 x i8], [4 x i8]* @.str.4, i32 0, i32 2
	store i8 120, i8* %47
	%48 = getelementptr [4 x i8], [4 x i8]* @.str.4, i32 0, i32 3
	store i8 0, i8* %48
	%49 = getelementptr [4 x i8], [4 x i8]* @.str.4, i32 0, i32 0
	%50 = bitcast i8* %49 to i32*
	%51 = call i32 @rt_streqi(i32* %44, i32* %50)
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %53, label %54

dead38:
	br label %42

53:
	ret i32 1

54:
	%55 = load i32*, i32** %2
	%56 = bitcast i32* %55 to i32*
	%57 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 0
	store i8 99, i8* %57
	%58 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 1
	store i8 111, i8* %58
	%59 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 2
	store i8 109, i8* %59
	%60 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 3
	store i8 49, i8* %60
	%61 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 4
	store i8 0, i8* %61
	%62 = getelementptr [5 x i8], [5 x i8]* @.str.5, i32 0, i32 0
	%63 = bitcast i8* %62 to i32*
	%64 = call i32 @rt_streqi(i32* %56, i32* %63)
	%65 = icmp ne i32 %64, 0
	br i1 %65, label %66, label %67

dead39:
	br label %54

66:
	ret i32 1

67:
	%68 = load i32*, i32** %2
	%69 = bitcast i32* %68 to i32*
	%70 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 0
	store i8 99, i8* %70
	%71 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 1
	store i8 111, i8* %71
	%72 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 2
	store i8 109, i8* %72
	%73 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 3
	store i8 50, i8* %73
	%74 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 4
	store i8 0, i8* %74
	%75 = getelementptr [5 x i8], [5 x i8]* @.str.6, i32 0, i32 0
	%76 = bitcast i8* %75 to i32*
	%77 = call i32 @rt_streqi(i32* %69, i32* %76)
	%78 = icmp ne i32 %77, 0
	br i1 %78, label %79, label %80

dead40:
	br label %67

79:
	ret i32 1

80:
	%81 = load i32*, i32** %2
	%82 = bitcast i32* %81 to i32*
	%83 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 0
	store i8 99, i8* %83
	%84 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 1
	store i8 111, i8* %84
	%85 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 2
	store i8 109, i8* %85
	%86 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 3
	store i8 51, i8* %86
	%87 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 4
	store i8 0, i8* %87
	%88 = getelementptr [5 x i8], [5 x i8]* @.str.7, i32 0, i32 0
	%89 = bitcast i8* %88 to i32*
	%90 = call i32 @rt_streqi(i32* %82, i32* %89)
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %92, label %93

dead41:
	br label %80

92:
	ret i32 1

93:
	%94 = load i32*, i32** %2
	%95 = bitcast i32* %94 to i32*
	%96 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 0
	store i8 99, i8* %96
	%97 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 1
	store i8 111, i8* %97
	%98 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 2
	store i8 109, i8* %98
	%99 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 3
	store i8 52, i8* %99
	%100 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 4
	store i8 0, i8* %100
	%101 = getelementptr [5 x i8], [5 x i8]* @.str.8, i32 0, i32 0
	%102 = bitcast i8* %101 to i32*
	%103 = call i32 @rt_streqi(i32* %95, i32* %102)
	%104 = icmp ne i32 %103, 0
	br i1 %104, label %105, label %106

dead42:
	br label %93

105:
	ret i32 1

106:
	%107 = load i32*, i32** %2
	%108 = bitcast i32* %107 to i32*
	%109 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 0
	store i8 99, i8* %109
	%110 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 1
	store i8 111, i8* %110
	%111 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 2
	store i8 109, i8* %111
	%112 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 3
	store i8 53, i8* %112
	%113 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 4
	store i8 0, i8* %113
	%114 = getelementptr [5 x i8], [5 x i8]* @.str.9, i32 0, i32 0
	%115 = bitcast i8* %114 to i32*
	%116 = call i32 @rt_streqi(i32* %108, i32* %115)
	%117 = icmp ne i32 %116, 0
	br i1 %117, label %118, label %119

dead43:
	br label %106

118:
	ret i32 1

119:
	%120 = load i32*, i32** %2
	%121 = bitcast i32* %120 to i32*
	%122 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 0
	store i8 99, i8* %122
	%123 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 1
	store i8 111, i8* %123
	%124 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 2
	store i8 109, i8* %124
	%125 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 3
	store i8 54, i8* %125
	%126 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 4
	store i8 0, i8* %126
	%127 = getelementptr [5 x i8], [5 x i8]* @.str.10, i32 0, i32 0
	%128 = bitcast i8* %127 to i32*
	%129 = call i32 @rt_streqi(i32* %121, i32* %128)
	%130 = icmp ne i32 %129, 0
	br i1 %130, label %131, label %132

dead44:
	br label %119

131:
	ret i32 1

132:
	%133 = load i32*, i32** %2
	%134 = bitcast i32* %133 to i32*
	%135 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 0
	store i8 99, i8* %135
	%136 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 1
	store i8 111, i8* %136
	%137 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 2
	store i8 109, i8* %137
	%138 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 3
	store i8 55, i8* %138
	%139 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 4
	store i8 0, i8* %139
	%140 = getelementptr [5 x i8], [5 x i8]* @.str.11, i32 0, i32 0
	%141 = bitcast i8* %140 to i32*
	%142 = call i32 @rt_streqi(i32* %134, i32* %141)
	%143 = icmp ne i32 %142, 0
	br i1 %143, label %144, label %145

dead45:
	br label %132

144:
	ret i32 1

145:
	%146 = load i32*, i32** %2
	%147 = bitcast i32* %146 to i32*
	%148 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 0
	store i8 99, i8* %148
	%149 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 1
	store i8 111, i8* %149
	%150 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 2
	store i8 109, i8* %150
	%151 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 3
	store i8 56, i8* %151
	%152 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 4
	store i8 0, i8* %152
	%153 = getelementptr [5 x i8], [5 x i8]* @.str.12, i32 0, i32 0
	%154 = bitcast i8* %153 to i32*
	%155 = call i32 @rt_streqi(i32* %147, i32* %154)
	%156 = icmp ne i32 %155, 0
	br i1 %156, label %157, label %158

dead46:
	br label %145

157:
	ret i32 1

158:
	%159 = load i32*, i32** %2
	%160 = bitcast i32* %159 to i32*
	%161 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 0
	store i8 99, i8* %161
	%162 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 1
	store i8 111, i8* %162
	%163 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 2
	store i8 109, i8* %163
	%164 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 3
	store i8 57, i8* %164
	%165 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 4
	store i8 0, i8* %165
	%166 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 0
	%167 = bitcast i8* %166 to i32*
	%168 = call i32 @rt_streqi(i32* %160, i32* %167)
	%169 = icmp ne i32 %168, 0
	br i1 %169, label %170, label %171

dead47:
	br label %158

170:
	ret i32 1

171:
	%172 = load i32*, i32** %2
	%173 = bitcast i32* %172 to i32*
	%174 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 0
	store i8 108, i8* %174
	%175 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 1
	store i8 112, i8* %175
	%176 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 2
	store i8 116, i8* %176
	%177 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 3
	store i8 49, i8* %177
	%178 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 4
	store i8 0, i8* %178
	%179 = getelementptr [5 x i8], [5 x i8]* @.str.14, i32 0, i32 0
	%180 = bitcast i8* %179 to i32*
	%181 = call i32 @rt_streqi(i32* %173, i32* %180)
	%182 = icmp ne i32 %181, 0
	br i1 %182, label %183, label %184

dead48:
	br label %171

183:
	ret i32 1

184:
	%185 = load i32*, i32** %2
	%186 = bitcast i32* %185 to i32*
	%187 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 0
	store i8 108, i8* %187
	%188 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 1
	store i8 112, i8* %188
	%189 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 2
	store i8 116, i8* %189
	%190 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 3
	store i8 50, i8* %190
	%191 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 4
	store i8 0, i8* %191
	%192 = getelementptr [5 x i8], [5 x i8]* @.str.15, i32 0, i32 0
	%193 = bitcast i8* %192 to i32*
	%194 = call i32 @rt_streqi(i32* %186, i32* %193)
	%195 = icmp ne i32 %194, 0
	br i1 %195, label %196, label %197

dead49:
	br label %184

196:
	ret i32 1

197:
	%198 = load i32*, i32** %2
	%199 = bitcast i32* %198 to i32*
	%200 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 0
	store i8 108, i8* %200
	%201 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 1
	store i8 112, i8* %201
	%202 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 2
	store i8 116, i8* %202
	%203 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 3
	store i8 51, i8* %203
	%204 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 4
	store i8 0, i8* %204
	%205 = getelementptr [5 x i8], [5 x i8]* @.str.16, i32 0, i32 0
	%206 = bitcast i8* %205 to i32*
	%207 = call i32 @rt_streqi(i32* %199, i32* %206)
	%208 = icmp ne i32 %207, 0
	br i1 %208, label %209, label %210

dead50:
	br label %197

209:
	ret i32 1

210:
	%211 = load i32*, i32** %2
	%212 = bitcast i32* %211 to i32*
	%213 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 0
	store i8 108, i8* %213
	%214 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 1
	store i8 112, i8* %214
	%215 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 2
	store i8 116, i8* %215
	%216 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 3
	store i8 52, i8* %216
	%217 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 4
	store i8 0, i8* %217
	%218 = getelementptr [5 x i8], [5 x i8]* @.str.17, i32 0, i32 0
	%219 = bitcast i8* %218 to i32*
	%220 = call i32 @rt_streqi(i32* %212, i32* %219)
	%221 = icmp ne i32 %220, 0
	br i1 %221, label %222, label %223

dead51:
	br label %210

222:
	ret i32 1

223:
	%224 = load i32*, i32** %2
	%225 = bitcast i32* %224 to i32*
	%226 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 0
	store i8 108, i8* %226
	%227 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 1
	store i8 112, i8* %227
	%228 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 2
	store i8 116, i8* %228
	%229 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 3
	store i8 53, i8* %229
	%230 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 4
	store i8 0, i8* %230
	%231 = getelementptr [5 x i8], [5 x i8]* @.str.18, i32 0, i32 0
	%232 = bitcast i8* %231 to i32*
	%233 = call i32 @rt_streqi(i32* %225, i32* %232)
	%234 = icmp ne i32 %233, 0
	br i1 %234, label %235, label %236

dead52:
	br label %223

235:
	ret i32 1

236:
	%237 = load i32*, i32** %2
	%238 = bitcast i32* %237 to i32*
	%239 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 0
	store i8 108, i8* %239
	%240 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 1
	store i8 112, i8* %240
	%241 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 2
	store i8 116, i8* %241
	%242 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 3
	store i8 54, i8* %242
	%243 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 4
	store i8 0, i8* %243
	%244 = getelementptr [5 x i8], [5 x i8]* @.str.19, i32 0, i32 0
	%245 = bitcast i8* %244 to i32*
	%246 = call i32 @rt_streqi(i32* %238, i32* %245)
	%247 = icmp ne i32 %246, 0
	br i1 %247, label %248, label %249

dead53:
	br label %236

248:
	ret i32 1

249:
	%250 = load i32*, i32** %2
	%251 = bitcast i32* %250 to i32*
	%252 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 0
	store i8 108, i8* %252
	%253 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 1
	store i8 112, i8* %253
	%254 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 2
	store i8 116, i8* %254
	%255 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 3
	store i8 55, i8* %255
	%256 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 4
	store i8 0, i8* %256
	%257 = getelementptr [5 x i8], [5 x i8]* @.str.20, i32 0, i32 0
	%258 = bitcast i8* %257 to i32*
	%259 = call i32 @rt_streqi(i32* %251, i32* %258)
	%260 = icmp ne i32 %259, 0
	br i1 %260, label %261, label %262

dead54:
	br label %249

261:
	ret i32 1

262:
	%263 = load i32*, i32** %2
	%264 = bitcast i32* %263 to i32*
	%265 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 0
	store i8 108, i8* %265
	%266 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 1
	store i8 112, i8* %266
	%267 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 2
	store i8 116, i8* %267
	%268 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 3
	store i8 56, i8* %268
	%269 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 4
	store i8 0, i8* %269
	%270 = getelementptr [5 x i8], [5 x i8]* @.str.21, i32 0, i32 0
	%271 = bitcast i8* %270 to i32*
	%272 = call i32 @rt_streqi(i32* %264, i32* %271)
	%273 = icmp ne i32 %272, 0
	br i1 %273, label %274, label %275

dead55:
	br label %262

274:
	ret i32 1

275:
	%276 = load i32*, i32** %2
	%277 = bitcast i32* %276 to i32*
	%278 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 0
	store i8 108, i8* %278
	%279 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 1
	store i8 112, i8* %279
	%280 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 2
	store i8 116, i8* %280
	%281 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 3
	store i8 57, i8* %281
	%282 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 4
	store i8 0, i8* %282
	%283 = getelementptr [5 x i8], [5 x i8]* @.str.22, i32 0, i32 0
	%284 = bitcast i8* %283 to i32*
	%285 = call i32 @rt_streqi(i32* %277, i32* %284)
	%286 = icmp ne i32 %285, 0
	br i1 %286, label %287, label %288

dead56:
	br label %275

287:
	ret i32 1

288:
	%289 = load i32*, i32** %2
	%290 = bitcast i32* %289 to i32*
	%291 = getelementptr [3 x i8], [3 x i8]* @.str.23, i32 0, i32 0
	store i8 99, i8* %291
	%292 = getelementptr [3 x i8], [3 x i8]* @.str.23, i32 0, i32 1
	store i8 58, i8* %292
	%293 = getelementptr [3 x i8], [3 x i8]* @.str.23, i32 0, i32 2
	store i8 0, i8* %293
	%294 = getelementptr [3 x i8], [3 x i8]* @.str.23, i32 0, i32 0
	%295 = bitcast i8* %294 to i32*
	%296 = call i32 @rt_streqi(i32* %290, i32* %295)
	%297 = icmp ne i32 %296, 0
	br i1 %297, label %298, label %299

dead57:
	br label %288

298:
	ret i32 2

299:
	%300 = load i32*, i32** %2
	%301 = bitcast i32* %300 to i32*
	%302 = getelementptr [4 x i8], [4 x i8]* @.str.24, i32 0, i32 0
	store i8 99, i8* %302
	%303 = getelementptr [4 x i8], [4 x i8]* @.str.24, i32 0, i32 1
	store i8 58, i8* %303
	%304 = getelementptr [4 x i8], [4 x i8]* @.str.24, i32 0, i32 2
	store i8 92, i8* %304
	%305 = getelementptr [4 x i8], [4 x i8]* @.str.24, i32 0, i32 3
	store i8 0, i8* %305
	%306 = getelementptr [4 x i8], [4 x i8]* @.str.24, i32 0, i32 0
	%307 = bitcast i8* %306 to i32*
	%308 = call i32 @rt_streqi(i32* %301, i32* %307)
	%309 = icmp ne i32 %308, 0
	br i1 %309, label %310, label %311

dead58:
	br label %299

310:
	ret i32 2

311:
	%312 = load i32*, i32** %2
	%313 = bitcast i32* %312 to i32*
	%314 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 0
	store i8 99, i8* %314
	%315 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 1
	store i8 58, i8* %315
	%316 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 2
	store i8 92, i8* %316
	%317 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 3
	store i8 119, i8* %317
	%318 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 4
	store i8 105, i8* %318
	%319 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 5
	store i8 110, i8* %319
	%320 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 6
	store i8 100, i8* %320
	%321 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 7
	store i8 111, i8* %321
	%322 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 8
	store i8 119, i8* %322
	%323 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 9
	store i8 115, i8* %323
	%324 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 10
	store i8 0, i8* %324
	%325 = getelementptr [11 x i8], [11 x i8]* @.str.25, i32 0, i32 0
	%326 = bitcast i8* %325 to i32*
	%327 = call i32 @rt_streqi(i32* %313, i32* %326)
	%328 = icmp ne i32 %327, 0
	br i1 %328, label %329, label %330

dead59:
	br label %311

329:
	ret i32 2

330:
	%331 = load i32*, i32** %2
	%332 = bitcast i32* %331 to i32*
	%333 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 0
	store i8 99, i8* %333
	%334 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 1
	store i8 58, i8* %334
	%335 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 2
	store i8 92, i8* %335
	%336 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 3
	store i8 119, i8* %336
	%337 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 4
	store i8 105, i8* %337
	%338 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 5
	store i8 110, i8* %338
	%339 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 6
	store i8 100, i8* %339
	%340 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 7
	store i8 111, i8* %340
	%341 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 8
	store i8 119, i8* %341
	%342 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 9
	store i8 115, i8* %342
	%343 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 10
	store i8 92, i8* %343
	%344 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 11
	store i8 0, i8* %344
	%345 = getelementptr [12 x i8], [12 x i8]* @.str.26, i32 0, i32 0
	%346 = bitcast i8* %345 to i32*
	%347 = call i32 @rt_streqi(i32* %332, i32* %346)
	%348 = icmp ne i32 %347, 0
	br i1 %348, label %349, label %350

dead60:
	br label %330

349:
	ret i32 2

350:
	%351 = load i32*, i32** %2
	%352 = bitcast i32* %351 to i32*
	%353 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 0
	store i8 99, i8* %353
	%354 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 1
	store i8 58, i8* %354
	%355 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 2
	store i8 92, i8* %355
	%356 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 3
	store i8 119, i8* %356
	%357 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 4
	store i8 105, i8* %357
	%358 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 5
	store i8 110, i8* %358
	%359 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 6
	store i8 100, i8* %359
	%360 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 7
	store i8 111, i8* %360
	%361 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 8
	store i8 119, i8* %361
	%362 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 9
	store i8 115, i8* %362
	%363 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 10
	store i8 92, i8* %363
	%364 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 11
	store i8 115, i8* %364
	%365 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 12
	store i8 121, i8* %365
	%366 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 13
	store i8 115, i8* %366
	%367 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 14
	store i8 116, i8* %367
	%368 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 15
	store i8 101, i8* %368
	%369 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 16
	store i8 109, i8* %369
	%370 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 17
	store i8 51, i8* %370
	%371 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 18
	store i8 50, i8* %371
	%372 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 19
	store i8 0, i8* %372
	%373 = getelementptr [20 x i8], [20 x i8]* @.str.27, i32 0, i32 0
	%374 = bitcast i8* %373 to i32*
	%375 = call i32 @rt_streqi(i32* %352, i32* %374)
	%376 = icmp ne i32 %375, 0
	br i1 %376, label %377, label %378

dead61:
	br label %350

377:
	ret i32 2

378:
	%379 = load i32*, i32** %2
	%380 = bitcast i32* %379 to i32*
	%381 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 0
	store i8 99, i8* %381
	%382 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 1
	store i8 58, i8* %382
	%383 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 2
	store i8 92, i8* %383
	%384 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 3
	store i8 119, i8* %384
	%385 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 4
	store i8 105, i8* %385
	%386 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 5
	store i8 110, i8* %386
	%387 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 6
	store i8 100, i8* %387
	%388 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 7
	store i8 111, i8* %388
	%389 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 8
	store i8 119, i8* %389
	%390 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 9
	store i8 115, i8* %390
	%391 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 10
	store i8 92, i8* %391
	%392 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 11
	store i8 115, i8* %392
	%393 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 12
	store i8 121, i8* %393
	%394 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 13
	store i8 115, i8* %394
	%395 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 14
	store i8 116, i8* %395
	%396 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 15
	store i8 101, i8* %396
	%397 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 16
	store i8 109, i8* %397
	%398 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 17
	store i8 51, i8* %398
	%399 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 18
	store i8 50, i8* %399
	%400 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 19
	store i8 92, i8* %400
	%401 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 20
	store i8 0, i8* %401
	%402 = getelementptr [21 x i8], [21 x i8]* @.str.28, i32 0, i32 0
	%403 = bitcast i8* %402 to i32*
	%404 = call i32 @rt_streqi(i32* %380, i32* %403)
	%405 = icmp ne i32 %404, 0
	br i1 %405, label %406, label %407

dead62:
	br label %378

406:
	ret i32 2

407:
	%408 = load i32*, i32** %2
	%409 = bitcast i32* %408 to i32*
	%410 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 0
	store i8 99, i8* %410
	%411 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 1
	store i8 58, i8* %411
	%412 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 2
	store i8 92, i8* %412
	%413 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 3
	store i8 117, i8* %413
	%414 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 4
	store i8 115, i8* %414
	%415 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 5
	store i8 101, i8* %415
	%416 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 6
	store i8 114, i8* %416
	%417 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 7
	store i8 115, i8* %417
	%418 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 8
	store i8 0, i8* %418
	%419 = getelementptr [9 x i8], [9 x i8]* @.str.29, i32 0, i32 0
	%420 = bitcast i8* %419 to i32*
	%421 = call i32 @rt_streqi(i32* %409, i32* %420)
	%422 = icmp ne i32 %421, 0
	br i1 %422, label %423, label %424

dead63:
	br label %407

423:
	ret i32 2

424:
	%425 = load i32*, i32** %2
	%426 = bitcast i32* %425 to i32*
	%427 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 0
	store i8 99, i8* %427
	%428 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 1
	store i8 58, i8* %428
	%429 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 2
	store i8 92, i8* %429
	%430 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 3
	store i8 117, i8* %430
	%431 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 4
	store i8 115, i8* %431
	%432 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 5
	store i8 101, i8* %432
	%433 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 6
	store i8 114, i8* %433
	%434 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 7
	store i8 115, i8* %434
	%435 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 8
	store i8 92, i8* %435
	%436 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 9
	store i8 0, i8* %436
	%437 = getelementptr [10 x i8], [10 x i8]* @.str.30, i32 0, i32 0
	%438 = bitcast i8* %437 to i32*
	%439 = call i32 @rt_streqi(i32* %426, i32* %438)
	%440 = icmp ne i32 %439, 0
	br i1 %440, label %441, label %442

dead64:
	br label %424

441:
	ret i32 2

442:
	%443 = load i32*, i32** %2
	%444 = bitcast i32* %443 to i32*
	%445 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 0
	store i8 99, i8* %445
	%446 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 1
	store i8 58, i8* %446
	%447 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 2
	store i8 92, i8* %447
	%448 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 3
	store i8 112, i8* %448
	%449 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 4
	store i8 114, i8* %449
	%450 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 5
	store i8 111, i8* %450
	%451 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 6
	store i8 103, i8* %451
	%452 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 7
	store i8 114, i8* %452
	%453 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 8
	store i8 97, i8* %453
	%454 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 9
	store i8 109, i8* %454
	%455 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 10
	store i8 32, i8* %455
	%456 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 11
	store i8 102, i8* %456
	%457 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 12
	store i8 105, i8* %457
	%458 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 13
	store i8 108, i8* %458
	%459 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 14
	store i8 101, i8* %459
	%460 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 15
	store i8 115, i8* %460
	%461 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 16
	store i8 0, i8* %461
	%462 = getelementptr [17 x i8], [17 x i8]* @.str.31, i32 0, i32 0
	%463 = bitcast i8* %462 to i32*
	%464 = call i32 @rt_streqi(i32* %444, i32* %463)
	%465 = icmp ne i32 %464, 0
	br i1 %465, label %466, label %467

dead65:
	br label %442

466:
	ret i32 2

467:
	%468 = load i32*, i32** %2
	%469 = bitcast i32* %468 to i32*
	%470 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 0
	store i8 99, i8* %470
	%471 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 1
	store i8 58, i8* %471
	%472 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 2
	store i8 92, i8* %472
	%473 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 3
	store i8 112, i8* %473
	%474 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 4
	store i8 114, i8* %474
	%475 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 5
	store i8 111, i8* %475
	%476 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 6
	store i8 103, i8* %476
	%477 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 7
	store i8 114, i8* %477
	%478 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 8
	store i8 97, i8* %478
	%479 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 9
	store i8 109, i8* %479
	%480 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 10
	store i8 32, i8* %480
	%481 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 11
	store i8 102, i8* %481
	%482 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 12
	store i8 105, i8* %482
	%483 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 13
	store i8 108, i8* %483
	%484 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 14
	store i8 101, i8* %484
	%485 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 15
	store i8 115, i8* %485
	%486 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 16
	store i8 92, i8* %486
	%487 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 17
	store i8 0, i8* %487
	%488 = getelementptr [18 x i8], [18 x i8]* @.str.32, i32 0, i32 0
	%489 = bitcast i8* %488 to i32*
	%490 = call i32 @rt_streqi(i32* %469, i32* %489)
	%491 = icmp ne i32 %490, 0
	br i1 %491, label %492, label %493

dead66:
	br label %467

492:
	ret i32 2

493:
	%494 = load i32*, i32** %2
	%495 = bitcast i32* %494 to i32*
	%496 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 0
	store i8 99, i8* %496
	%497 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 1
	store i8 58, i8* %497
	%498 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 2
	store i8 92, i8* %498
	%499 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 3
	store i8 112, i8* %499
	%500 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 4
	store i8 114, i8* %500
	%501 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 5
	store i8 111, i8* %501
	%502 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 6
	store i8 103, i8* %502
	%503 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 7
	store i8 114, i8* %503
	%504 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 8
	store i8 97, i8* %504
	%505 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 9
	store i8 109, i8* %505
	%506 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 10
	store i8 100, i8* %506
	%507 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 11
	store i8 97, i8* %507
	%508 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 12
	store i8 116, i8* %508
	%509 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 13
	store i8 97, i8* %509
	%510 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 14
	store i8 0, i8* %510
	%511 = getelementptr [15 x i8], [15 x i8]* @.str.33, i32 0, i32 0
	%512 = bitcast i8* %511 to i32*
	%513 = call i32 @rt_streqi(i32* %495, i32* %512)
	%514 = icmp ne i32 %513, 0
	br i1 %514, label %515, label %516

dead67:
	br label %493

515:
	ret i32 2

516:
	%517 = load i32*, i32** %2
	%518 = bitcast i32* %517 to i32*
	%519 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 0
	store i8 99, i8* %519
	%520 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 1
	store i8 58, i8* %520
	%521 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 2
	store i8 92, i8* %521
	%522 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 3
	store i8 112, i8* %522
	%523 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 4
	store i8 114, i8* %523
	%524 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 5
	store i8 111, i8* %524
	%525 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 6
	store i8 103, i8* %525
	%526 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 7
	store i8 114, i8* %526
	%527 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 8
	store i8 97, i8* %527
	%528 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 9
	store i8 109, i8* %528
	%529 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 10
	store i8 100, i8* %529
	%530 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 11
	store i8 97, i8* %530
	%531 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 12
	store i8 116, i8* %531
	%532 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 13
	store i8 97, i8* %532
	%533 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 14
	store i8 92, i8* %533
	%534 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 15
	store i8 0, i8* %534
	%535 = getelementptr [16 x i8], [16 x i8]* @.str.34, i32 0, i32 0
	%536 = bitcast i8* %535 to i32*
	%537 = call i32 @rt_streqi(i32* %518, i32* %536)
	%538 = icmp ne i32 %537, 0
	br i1 %538, label %539, label %540

dead68:
	br label %516

539:
	ret i32 2

540:
	ret i32 0

dead69:
	br label %540

dead70:
	ret i32 0
}

define i32 @rt_isdelim(i32 %0, i32* %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = icmp ne i32 1, 0
	br i1 %6, label %7, label %18

7:
	%8 = alloca i32
	%9 = load i32, i32* %4
	%10 = load i32*, i32** %3
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = load i8, i8* %11
	%13 = sext i8 %12 to i32
	store i32 %13, i32* %8
	%14 = load i32, i32* %8
	%15 = icmp eq i32 %14, 0
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %19, label %20

18:
	ret i32 0

19:
	ret i32 0

20:
	%21 = load i32, i32* %8
	%22 = load i32, i32* %2
	%23 = icmp eq i32 %21, %22
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %27

dead71:
	br label %20

26:
	ret i32 1

27:
	%28 = load i32, i32* %4
	%29 = add i32 %28, 1
	store i32 %29, i32* %4
	br label %5

dead72:
	br label %27

dead73:
	ret i32 0
}

define i32 @rt_tokb(i32* %0, i32* %1, i32 %2, i32 %3) {
entry:
	%4 = alloca i32*
	store i32* %0, i32** %4
	%5 = alloca i32*
	store i32* %1, i32** %5
	%6 = alloca i32
	store i32 %2, i32* %6
	%7 = alloca i32
	store i32 %3, i32* %7
	%8 = alloca i32
	store i32 0, i32* %8
	%9 = alloca i32
	store i32 0, i32* %9
	%10 = alloca i32
	store i32 0, i32* %10
	br label %11

11:
	%12 = icmp ne i32 1, 0
	br i1 %12, label %13, label %14

13:
	br label %15

14:
	ret i32 -1

15:
	%16 = load i32, i32* %8
	%17 = load i32*, i32** %4
	%18 = getelementptr i8, i32* %17, i32 %16
	%19 = load i8, i8* %18
	%20 = sext i8 %19 to i32
	%21 = load i32*, i32** %5
	%22 = bitcast i32* %21 to i32*
	%23 = call i32 @rt_isdelim(i32 %20, i32* %22)
	%24 = icmp ne i32 %23, 0
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %27, label %30

27:
	%28 = load i32, i32* %8
	%29 = add i32 %28, 1
	store i32 %29, i32* %8
	br label %15

30:
	%31 = load i32, i32* %8
	%32 = load i32*, i32** %4
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = load i8, i8* %33
	%35 = sext i8 %34 to i32
	%36 = icmp eq i32 %35, 0
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %39, label %40

39:
	ret i32 -1

40:
	%41 = load i32, i32* %8
	store i32 %41, i32* %10
	br label %42

dead74:
	br label %40

42:
	%43 = load i32, i32* %8
	%44 = load i32*, i32** %4
	%45 = getelementptr i8, i32* %44, i32 %43
	%46 = load i8, i8* %45
	%47 = sext i8 %46 to i32
	%48 = icmp ne i32 %47, 0
	%49 = zext i1 %48 to i32
	%50 = icmp ne i32 %49, 0
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %64, label %77

53:
	%54 = load i32, i32* %8
	%55 = add i32 %54, 1
	store i32 %55, i32* %8
	br label %42

56:
	%57 = load i32, i32* %9
	%58 = add i32 %57, 1
	store i32 %58, i32* %9
	%59 = load i32, i32* %9
	%60 = load i32, i32* %6
	%61 = icmp eq i32 %59, %60
	%62 = zext i1 %61 to i32
	%63 = icmp ne i32 %62, 0
	br i1 %63, label %80, label %85

64:
	%65 = load i32, i32* %8
	%66 = load i32*, i32** %4
	%67 = getelementptr i8, i32* %66, i32 %65
	%68 = load i8, i8* %67
	%69 = sext i8 %68 to i32
	%70 = load i32*, i32** %5
	%71 = bitcast i32* %70 to i32*
	%72 = call i32 @rt_isdelim(i32 %69, i32* %71)
	%73 = icmp eq i32 %72, 0
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	%76 = zext i1 %75 to i32
	br label %77

77:
	%78 = phi i32 [ %51, %42 ], [ %76, %64 ]
	%79 = icmp ne i32 %78, 0
	br i1 %79, label %53, label %56

80:
	%81 = load i32, i32* %7
	%82 = icmp eq i32 %81, 2
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %86, label %87

85:
	br label %11

86:
	br label %92

87:
	%88 = load i32, i32* %7
	%89 = icmp eq i32 %88, 0
	%90 = zext i1 %89 to i32
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %109, label %111

92:
	%93 = load i32, i32* %8
	%94 = load i32*, i32** %4
	%95 = getelementptr i8, i32* %94, i32 %93
	%96 = load i8, i8* %95
	%97 = sext i8 %96 to i32
	%98 = load i32*, i32** %5
	%99 = bitcast i32* %98 to i32*
	%100 = call i32 @rt_isdelim(i32 %97, i32* %99)
	%101 = icmp ne i32 %100, 0
	%102 = zext i1 %101 to i32
	%103 = icmp ne i32 %102, 0
	br i1 %103, label %104, label %107

104:
	%105 = load i32, i32* %8
	%106 = add i32 %105, 1
	store i32 %106, i32* %8
	br label %92

107:
	%108 = load i32, i32* %8
	ret i32 %108

dead75:
	br label %87

109:
	%110 = load i32, i32* %10
	br label %113

111:
	%112 = load i32, i32* %8
	br label %113

113:
	%114 = phi i32 [ %110, %109 ], [ %112, %111 ]
	ret i32 %114

dead76:
	br label %85

dead77:
	ret i32 0
}

define i32 @rt_nlines(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = alloca i32
	store i32 0, i32* %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = load i32, i32* %2
	%7 = load i32*, i32** %1
	%8 = getelementptr i8, i32* %7, i32 %6
	%9 = load i8, i8* %8
	%10 = sext i8 %9 to i32
	%11 = icmp ne i32 %10, 0
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	br i1 %13, label %14, label %23

14:
	%15 = load i32, i32* %2
	%16 = load i32*, i32** %1
	%17 = getelementptr i8, i32* %16, i32 %15
	%18 = load i8, i8* %17
	%19 = sext i8 %18 to i32
	%20 = icmp eq i32 %19, 10
	%21 = zext i1 %20 to i32
	%22 = icmp ne i32 %21, 0
	br i1 %22, label %27, label %33

23:
	%24 = load i32, i32* %3
	%25 = load i32, i32* %4
	%26 = add i32 %24, %25
	ret i32 %26

27:
	%28 = load i32, i32* %3
	%29 = add i32 %28, 1
	store i32 %29, i32* %3
	store i32 0, i32* %4
	br label %30

30:
	%31 = load i32, i32* %2
	%32 = add i32 %31, 1
	store i32 %32, i32* %2
	br label %5

33:
	store i32 1, i32* %4
	br label %30

dead78:
	ret i32 0
}

define i32* @rt_lineat(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = call i32 @rt_strlen(i32* %6)
	store i32 %7, i32* %4
	%8 = alloca i32
	store i32 0, i32* %8
	%9 = alloca i32
	store i32 0, i32* %9
	%10 = alloca i32
	store i32 0, i32* %10
	%11 = alloca i32
	store i32 -1, i32* %11
	br label %12

12:
	%13 = load i32, i32* %8
	%14 = load i32, i32* %4
	%15 = icmp slt i32 %13, %14
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %18, label %27

18:
	%19 = load i32, i32* %8
	%20 = load i32*, i32** %2
	%21 = getelementptr i8, i32* %20, i32 %19
	%22 = load i8, i8* %21
	%23 = sext i8 %22 to i32
	%24 = icmp eq i32 %23, 10
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %32, label %39

27:
	%28 = load i32, i32* %11
	%29 = icmp slt i32 %28, 0
	%30 = zext i1 %29 to i32
	%31 = icmp ne i32 %30, 0
	br i1 %31, label %53, label %59

32:
	%33 = load i32, i32* %9
	%34 = load i32, i32* %3
	%35 = icmp eq i32 %33, %34
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %42, label %46

38:
	br label %12

39:
	%40 = load i32, i32* %8
	%41 = add i32 %40, 1
	store i32 %41, i32* %8
	br label %38

42:
	%43 = load i32, i32* %8
	store i32 %43, i32* %11
	%44 = load i32, i32* %4
	store i32 %44, i32* %8
	br label %45

45:
	br label %38

46:
	%47 = load i32, i32* %9
	%48 = add i32 %47, 1
	store i32 %48, i32* %9
	%49 = load i32, i32* %8
	%50 = add i32 %49, 1
	store i32 %50, i32* %10
	%51 = load i32, i32* %8
	%52 = add i32 %51, 1
	store i32 %52, i32* %8
	br label %45

53:
	%54 = load i32, i32* %9
	%55 = load i32, i32* %3
	%56 = icmp ne i32 %54, %55
	%57 = zext i1 %56 to i32
	%58 = icmp ne i32 %57, 0
	br i1 %58, label %68, label %71

59:
	%60 = alloca i32
	%61 = load i32, i32* %11
	%62 = load i32, i32* %10
	%63 = icmp sgt i32 %61, %62
	%64 = zext i1 %63 to i32
	store i32 %64, i32* %60
	%65 = alloca i32
	%66 = load i32, i32* %60
	%67 = icmp ne i32 %66, 0
	br i1 %67, label %73, label %76

68:
	%69 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%70 = bitcast i8* %69 to i32*
	ret i32* %70

71:
	%72 = load i32, i32* %4
	store i32 %72, i32* %11
	br label %59

dead79:
	br label %71

73:
	%74 = load i32, i32* %11
	%75 = sub i32 %74, 1
	br label %78

76:
	%77 = load i32, i32* %10
	br label %78

78:
	%79 = phi i32 [ %75, %73 ], [ %77, %76 ]
	store i32 %79, i32* %65
	%80 = alloca i32
	%81 = load i32, i32* %60
	%82 = icmp ne i32 %81, 0
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %85, label %95

85:
	%86 = load i32, i32* %65
	%87 = load i32*, i32** %2
	%88 = getelementptr i8, i32* %87, i32 %86
	%89 = load i8, i8* %88
	%90 = sext i8 %89 to i32
	%91 = icmp eq i32 %90, 13
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	%94 = zext i1 %93 to i32
	br label %95

95:
	%96 = phi i32 [ %83, %78 ], [ %94, %85 ]
	store i32 %96, i32* %80
	%97 = load i32*, i32** %2
	%98 = bitcast i32* %97 to i32*
	%99 = load i32, i32* %10
	%100 = load i32, i32* %80
	%101 = icmp ne i32 %100, 0
	br i1 %101, label %102, label %105

102:
	%103 = load i32, i32* %11
	%104 = sub i32 %103, 1
	br label %107

105:
	%106 = load i32, i32* %11
	br label %107

107:
	%108 = phi i32 [ %104, %102 ], [ %106, %105 ]
	%109 = call i32* @rt_sub(i32* %98, i32 %99, i32 %108)
	%110 = bitcast i32* %109 to i32*
	ret i32* %110

dead80:
	ret i32* null
}

define i32* @rt_expand(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32*
	%7 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%8 = bitcast i8* %7 to i32*
	store i32* %8, i32** %6
	%9 = alloca i32
	store i32 0, i32* %9
	br label %10

10:
	%11 = load i32, i32* %9
	%12 = load i32, i32* %2
	%13 = icmp slt i32 %11, %12
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %16, label %26

16:
	%17 = alloca i32
	%18 = load i32, i32* %9
	%19 = load i32*, i32** %1
	%20 = getelementptr i8, i32* %19, i32 %18
	%21 = load i8, i8* %20
	%22 = sext i8 %21 to i32
	%23 = icmp eq i32 %22, 37
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %29, label %35

26:
	%27 = load i32*, i32** %6
	%28 = bitcast i32* %27 to i32*
	ret i32* %28

29:
	%30 = load i32*, i32** %1
	%31 = bitcast i32* %30 to i32*
	%32 = load i32, i32* %9
	%33 = add i32 %32, 1
	%34 = call i32 @rt_findch(i32* %31, i32 37, i32 %33)
	br label %36

35:
	br label %36

36:
	%37 = phi i32 [ %34, %29 ], [ -1, %35 ]
	store i32 %37, i32* %17
	%38 = load i32, i32* %17
	%39 = load i32, i32* %9
	%40 = add i32 %39, 1
	%41 = icmp sgt i32 %38, %40
	%42 = zext i1 %41 to i32
	%43 = icmp ne i32 %42, 0
	br i1 %43, label %44, label %61

44:
	%45 = load i32*, i32** %6
	%46 = bitcast i32* %45 to i32*
	%47 = load i32*, i32** %1
	%48 = bitcast i32* %47 to i32*
	%49 = load i32, i32* %9
	%50 = add i32 %49, 1
	%51 = load i32, i32* %17
	%52 = call i32* @rt_sub(i32* %48, i32 %50, i32 %51)
	%53 = bitcast i32* %52 to i32*
	%54 = call i32* @bat_lookup(i32* %53)
	%55 = bitcast i32* %54 to i32*
	%56 = call i32* @rt_strcat(i32* %46, i32* %55)
	%57 = bitcast i32* %56 to i32*
	store i32* %57, i32** %6
	%58 = load i32, i32* %17
	%59 = add i32 %58, 1
	store i32 %59, i32* %9
	br label %60

60:
	br label %10

61:
	%62 = load i32*, i32** %6
	%63 = bitcast i32* %62 to i32*
	%64 = load i32*, i32** %1
	%65 = bitcast i32* %64 to i32*
	%66 = load i32, i32* %9
	%67 = load i32, i32* %9
	%68 = add i32 %67, 1
	%69 = call i32* @rt_sub(i32* %65, i32 %66, i32 %68)
	%70 = bitcast i32* %69 to i32*
	%71 = call i32* @rt_strcat(i32* %63, i32* %70)
	%72 = bitcast i32* %71 to i32*
	store i32* %72, i32** %6
	%73 = load i32, i32* %9
	%74 = add i32 %73, 1
	store i32 %74, i32* %9
	br label %60

dead81:
	ret i32* null
}

declare i32* @bat_lookup(i32* %0)


define i32 @bat_shift(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = load i32, i32* @frame
	%5 = mul i32 %4, 10
	store i32 %5, i32* %2
	%6 = load i32, i32* %1
	store i32 %6, i32* %3
	br label %7

7:
	%8 = load i32, i32* %3
	%9 = icmp slt i32 %8, 9
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %26

12:
	%13 = load i32, i32* %2
	%14 = load i32, i32* %3
	%15 = add i32 %13, %14
	%16 = getelementptr [2560 x i32*], [2560 x i32*]* @args, i32 0, i32 %15
	%17 = load i32, i32* %2
	%18 = load i32, i32* %3
	%19 = add i32 %17, %18
	%20 = add i32 %19, 1
	%21 = getelementptr [2560 x i32*], [2560 x i32*]* @args, i32 0, i32 %20
	%22 = load i32*, i32** %21
	%23 = bitcast i32* %22 to i32*
	store i32* %23, i32** %16
	%24 = load i32, i32* %3
	%25 = add i32 %24, 1
	store i32 %25, i32* %3
	br label %7

26:
	%27 = load i32, i32* %2
	%28 = add i32 %27, 9
	%29 = getelementptr [2560 x i32*], [2560 x i32*]* @args, i32 0, i32 %28
	%30 = getelementptr [1 x i8], [1 x i8]* @EMPTY, i32 0, i32 0
	%31 = bitcast i8* %30 to i32*
	store i32* %31, i32** %29
	ret i32 0

dead83:
	ret i32 0
}


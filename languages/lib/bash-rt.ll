@last_status = global i32 zeroinitializer
@exited = global i32 zeroinitializer
@exit_code = global i32 zeroinitializer
@empty = global [1 x i8] zeroinitializer
@unset_marker = global [1 x i8] zeroinitializer
@gvars = global [1024 x i32*] zeroinitializer
@nvars = global i32 zeroinitializer
@rt_limit = global i32 zeroinitializer
@varnames = global [1024 x i32*] zeroinitializer
@ls_id = global [512 x i32] zeroinitializer
@ls_val = global [512 x i32*] zeroinitializer
@ls_top = global i32 zeroinitializer
@cap_buf = global [16 x i32*] zeroinitializer
@cap_len = global [16 x i32] zeroinitializer
@cap_sz = global [16 x i32] zeroinitializer
@cap_depth = global i32 zeroinitializer
@ss_save = global [8192 x i32*] zeroinitializer
@ss_depth = global i32 zeroinitializer
@stdin_buf = global i32* null
@stdin_pos = global i32 zeroinitializer
@opt_errexit = global i32 zeroinitializer
@opt_nounset = global i32 zeroinitializer
@opt_pipefail = global i32 zeroinitializer
@abort_flag = global i32 zeroinitializer
@cond_err = global i32 zeroinitializer
@sink_out = global i32 zeroinitializer
@sink_err = global i32 zeroinitializer
@argv = global [4096 x i32*] zeroinitializer
@argv_top = global i32 zeroinitializer
@frame_base = global i32 zeroinitializer
@frame_n = global i32 zeroinitializer
@arena = global [4194304 x i8] zeroinitializer
@arena_pos = global i32 zeroinitializer
@.str.1 = global [6 x i8] zeroinitializer
@.str.2 = global [6 x i8] zeroinitializer
@.str.3 = global [6 x i8] zeroinitializer
@.str.4 = global [6 x i8] zeroinitializer
@.str.5 = global [6 x i8] zeroinitializer
@.str.6 = global [6 x i8] zeroinitializer
@.str.7 = global [6 x i8] zeroinitializer
@.str.8 = global [6 x i8] zeroinitializer
@.str.9 = global [6 x i8] zeroinitializer
@.str.10 = global [6 x i8] zeroinitializer
@.str.11 = global [6 x i8] zeroinitializer
@.str.12 = global [7 x i8] zeroinitializer
@.str.13 = global [5 x i8] zeroinitializer
@.str.14 = global [6 x i8] zeroinitializer
@read_eof = global i32 zeroinitializer
@.str.15 = global [2 x i8] zeroinitializer
@.str.16 = global [2 x i8] zeroinitializer
@.str.17 = global [2 x i8] zeroinitializer
@.str.18 = global [2 x i8] zeroinitializer
@.str.19 = global [2 x i8] zeroinitializer
@.str.20 = global [2 x i8] zeroinitializer
@.str.21 = global [2 x i8] zeroinitializer
@.str.22 = global [2 x i8] zeroinitializer
@.str.23 = global [2 x i8] zeroinitializer
@arr_nm = global [4096 x i32*] zeroinitializer
@arr_k = global [4096 x i32*] zeroinitializer
@arr_v = global [4096 x i32*] zeroinitializer
@arr_n = global i32 zeroinitializer
@.str.24 = global [2 x i8] zeroinitializer
@.str.25 = global [2 x i8] zeroinitializer
@.str.26 = global [2 x i8] zeroinitializer
@.str.27 = global [2 x i8] zeroinitializer
@.str.28 = global [2 x i8] zeroinitializer
@.str.29 = global [2 x i8] zeroinitializer
@.str.30 = global [2 x i8] zeroinitializer
@.str.31 = global [10 x i8] zeroinitializer
@.str.32 = global [10 x i8] zeroinitializer
@.str.33 = global [9 x i8] zeroinitializer
@.str.34 = global [11 x i8] zeroinitializer
@.str.35 = global [12 x i8] zeroinitializer
@.str.36 = global [12 x i8] zeroinitializer
@.str.37 = global [2 x i8] zeroinitializer
@.str.38 = global [5 x i8] zeroinitializer
@.str.39 = global [5 x i8] zeroinitializer
@.str.40 = global [5 x i8] zeroinitializer
@.str.41 = global [5 x i8] zeroinitializer
@.str.42 = global [5 x i8] zeroinitializer
@.str.43 = global [5 x i8] zeroinitializer
@.str.44 = global [9 x i8] zeroinitializer
@re_prog = global [12288 x i32] zeroinitializer
@re_cls = global [2048 x i8] zeroinitializer
@re_slot = global [192 x i32] zeroinitializer
@re_pc = global i32 zeroinitializer
@re_pat = global i32* null
@re_pos = global i32 zeroinitializer
@re_ng = global i32 zeroinitializer
@re_ncls = global i32 zeroinitializer
@re_nmark = global i32 zeroinitializer
@re_depth = global i32 zeroinitializer
@re_err = global i32* null
@re_subj = global i32* null
@re_slen = global i32 zeroinitializer
@re_steps = global i32 zeroinitializer
@re_cstk = global [8192 x i32] zeroinitializer
@re_ctop = global i32 zeroinitializer
@re_fold = global i32 zeroinitializer
@re_bad = global i32 zeroinitializer
@.str.45 = global [18 x i8] zeroinitializer
@.str.46 = global [31 x i8] zeroinitializer
@.str.47 = global [24 x i8] zeroinitializer
@.str.48 = global [27 x i8] zeroinitializer
@.str.49 = global [37 x i8] zeroinitializer
@.str.50 = global [19 x i8] zeroinitializer
@.str.51 = global [18 x i8] zeroinitializer
@.str.52 = global [22 x i8] zeroinitializer
@.str.53 = global [29 x i8] zeroinitializer

define i32* @rt_bump(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32*
	%3 = load i32, i32* @arena_pos
	%4 = getelementptr [4194304 x i8], [4194304 x i8]* @arena, i32 0, i32 %3
	%5 = bitcast i8* %4 to i32*
	store i32* %5, i32** %2
	%6 = load i32, i32* @arena_pos
	%7 = load i32, i32* %1
	%8 = add i32 %6, %7
	store i32 %8, i32* @arena_pos
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

define i32 @rt_charlen(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = alloca i32
	store i32 0, i32* %3
	br label %4

4:
	%5 = load i32, i32* %2
	%6 = load i32*, i32** %1
	%7 = getelementptr i8, i32* %6, i32 %5
	%8 = load i8, i8* %7
	%9 = sext i8 %8 to i32
	%10 = icmp ne i32 %9, 0
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %13, label %26

13:
	%14 = alloca i32
	%15 = load i32, i32* %2
	%16 = load i32*, i32** %1
	%17 = getelementptr i8, i32* %16, i32 %15
	%18 = load i8, i8* %17
	%19 = sext i8 %18 to i32
	%20 = and i32 %19, 255
	store i32 %20, i32* %14
	%21 = load i32, i32* %14
	%22 = and i32 %21, 192
	%23 = icmp ne i32 %22, 128
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %28, label %31

26:
	%27 = load i32, i32* %3
	ret i32 %27

28:
	%29 = load i32, i32* %3
	%30 = add i32 %29, 1
	store i32 %30, i32* %3
	br label %31

31:
	%32 = load i32, i32* %2
	%33 = add i32 %32, 1
	store i32 %33, i32* %2
	br label %4

dead3:
	ret i32 0
}

define i32 @rt_charoff(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	store i32 0, i32* %4
	%5 = alloca i32
	store i32 0, i32* %5
	br label %6

6:
	%7 = load i32, i32* %4
	%8 = load i32*, i32** %2
	%9 = getelementptr i8, i32* %8, i32 %7
	%10 = load i8, i8* %9
	%11 = sext i8 %10 to i32
	%12 = icmp ne i32 %11, 0
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	br i1 %14, label %15, label %28

15:
	%16 = alloca i32
	%17 = load i32, i32* %4
	%18 = load i32*, i32** %2
	%19 = getelementptr i8, i32* %18, i32 %17
	%20 = load i8, i8* %19
	%21 = sext i8 %20 to i32
	%22 = and i32 %21, 255
	store i32 %22, i32* %16
	%23 = load i32, i32* %16
	%24 = and i32 %23, 192
	%25 = icmp ne i32 %24, 128
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	br i1 %27, label %30, label %36

28:
	%29 = load i32, i32* %4
	ret i32 %29

30:
	%31 = load i32, i32* %5
	%32 = load i32, i32* %3
	%33 = icmp sge i32 %31, %32
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %39, label %41

36:
	%37 = load i32, i32* %4
	%38 = add i32 %37, 1
	store i32 %38, i32* %4
	br label %6

39:
	%40 = load i32, i32* %4
	ret i32 %40

41:
	%42 = load i32, i32* %5
	%43 = add i32 %42, 1
	store i32 %43, i32* %5
	br label %36

dead4:
	br label %41

dead5:
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

dead6:
	br label %39

45:
	ret i32 1

46:
	%47 = load i32, i32* %4
	%48 = add i32 %47, 1
	store i32 %48, i32* %4
	br label %5

dead7:
	br label %46

dead8:
	ret i32 0
}

define i32 @rt_strcmp(i32* %0, i32* %1) {
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
	br i1 %6, label %7, label %27

7:
	%8 = alloca i32
	%9 = load i32, i32* %4
	%10 = load i32*, i32** %2
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = load i8, i8* %11
	%13 = sext i8 %12 to i32
	%14 = and i32 %13, 255
	store i32 %14, i32* %8
	%15 = alloca i32
	%16 = load i32, i32* %4
	%17 = load i32*, i32** %3
	%18 = getelementptr i8, i32* %17, i32 %16
	%19 = load i8, i8* %18
	%20 = sext i8 %19 to i32
	%21 = and i32 %20, 255
	store i32 %21, i32* %15
	%22 = load i32, i32* %8
	%23 = load i32, i32* %15
	%24 = icmp slt i32 %22, %23
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %28, label %29

27:
	ret i32 0

28:
	ret i32 -1

29:
	%30 = load i32, i32* %8
	%31 = load i32, i32* %15
	%32 = icmp sgt i32 %30, %31
	%33 = zext i1 %32 to i32
	%34 = icmp ne i32 %33, 0
	br i1 %34, label %35, label %36

dead9:
	br label %29

35:
	ret i32 1

36:
	%37 = load i32, i32* %8
	%38 = icmp eq i32 %37, 0
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	br i1 %40, label %41, label %42

dead10:
	br label %36

41:
	ret i32 0

42:
	%43 = load i32, i32* %4
	%44 = add i32 %43, 1
	store i32 %44, i32* %4
	br label %5

dead11:
	br label %42

dead12:
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
	%22 = load i32, i32* %19
	%23 = load i32*, i32** %2
	%24 = getelementptr i8, i32* %23, i32 %22
	%25 = load i8, i8* %24
	%26 = sext i8 %25 to i32
	%27 = icmp ne i32 %26, 0
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %30, label %48

30:
	%31 = load i32, i32* %20
	%32 = load i32*, i32** %12
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = load i32, i32* %19
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
	%44 = load i32, i32* %20
	%45 = add i32 %44, 1
	store i32 %45, i32* %20
	%46 = load i32, i32* %19
	%47 = add i32 %46, 1
	store i32 %47, i32* %19
	br label %21

48:
	store i32 0, i32* %19
	br label %49

49:
	%50 = load i32, i32* %19
	%51 = load i32*, i32** %3
	%52 = getelementptr i8, i32* %51, i32 %50
	%53 = load i8, i8* %52
	%54 = sext i8 %53 to i32
	%55 = icmp ne i32 %54, 0
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %58, label %76

58:
	%59 = load i32, i32* %20
	%60 = load i32*, i32** %12
	%61 = getelementptr i8, i32* %60, i32 %59
	%62 = load i32, i32* %19
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
	%72 = load i32, i32* %20
	%73 = add i32 %72, 1
	store i32 %73, i32* %20
	%74 = load i32, i32* %19
	%75 = add i32 %74, 1
	store i32 %75, i32* %19
	br label %49

76:
	%77 = load i32, i32* %20
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

dead13:
	ret i32* null
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

dead14:
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
	br i1 %63, label %64, label %81

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
	%74 = shl i32 %73, 24
	%75 = ashr i32 %74, 24
	%76 = trunc i32 %75 to i8
	store i8 %76, i8* %66
	%77 = load i32, i32* %58
	%78 = sub i32 %77, 1
	store i32 %78, i32* %58
	%79 = load i32, i32* %48
	%80 = sdiv i32 %79, 10
	store i32 %80, i32* %48
	br label %59

81:
	%82 = alloca i32
	store i32 0, i32* %82
	%83 = load i32, i32* %44
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %85, label %93

85:
	%86 = load i32*, i32** %19
	%87 = getelementptr i8, i32* %86, i32 0
	%88 = shl i32 45, 24
	%89 = ashr i32 %88, 24
	%90 = shl i32 %89, 24
	%91 = ashr i32 %90, 24
	%92 = trunc i32 %91 to i8
	store i8 %92, i8* %87
	store i32 1, i32* %82
	br label %93

93:
	%94 = alloca i32
	%95 = load i32, i32* %58
	%96 = add i32 %95, 1
	store i32 %96, i32* %94
	br label %97

97:
	%98 = load i32, i32* %94
	%99 = icmp sle i32 %98, 15
	%100 = zext i1 %99 to i32
	%101 = icmp ne i32 %100, 0
	br i1 %101, label %102, label %119

102:
	%103 = load i32, i32* %82
	%104 = load i32*, i32** %19
	%105 = getelementptr i8, i32* %104, i32 %103
	%106 = load i32, i32* %94
	%107 = getelementptr [16 x i8], [16 x i8]* %2, i32 0, i32 %106
	%108 = load i8, i8* %107
	%109 = sext i8 %108 to i32
	%110 = shl i32 %109, 24
	%111 = ashr i32 %110, 24
	%112 = shl i32 %111, 24
	%113 = ashr i32 %112, 24
	%114 = trunc i32 %113 to i8
	store i8 %114, i8* %105
	%115 = load i32, i32* %82
	%116 = add i32 %115, 1
	store i32 %116, i32* %82
	%117 = load i32, i32* %94
	%118 = add i32 %117, 1
	store i32 %118, i32* %94
	br label %97

119:
	%120 = load i32, i32* %82
	%121 = load i32*, i32** %19
	%122 = getelementptr i8, i32* %121, i32 %120
	%123 = shl i32 0, 24
	%124 = ashr i32 %123, 24
	%125 = shl i32 %124, 24
	%126 = ashr i32 %125, 24
	%127 = trunc i32 %126 to i8
	store i8 %127, i8* %122
	%128 = load i32*, i32** %19
	%129 = bitcast i32* %128 to i32*
	ret i32* %129

dead15:
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

dead16:
	ret i32 0
}

define i32 @rt_clsid(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 0
	store i8 97, i8* %5
	%6 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 1
	store i8 108, i8* %6
	%7 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 2
	store i8 112, i8* %7
	%8 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 3
	store i8 104, i8* %8
	%9 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 4
	store i8 97, i8* %9
	%10 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 5
	store i8 0, i8* %10
	%11 = getelementptr [6 x i8], [6 x i8]* @.str.1, i32 0, i32 0
	%12 = bitcast i8* %11 to i32*
	%13 = call i32 @rt_streq(i32* %4, i32* %12)
	%14 = icmp ne i32 %13, 0
	%15 = zext i1 %14 to i32
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %18

17:
	br label %20

18:
	%19 = load i32, i32* %2
	br label %20

20:
	%21 = phi i32 [ 1, %17 ], [ %19, %18 ]
	store i32 %21, i32* %2
	%22 = load i32*, i32** %1
	%23 = bitcast i32* %22 to i32*
	%24 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 0
	store i8 100, i8* %24
	%25 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 1
	store i8 105, i8* %25
	%26 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 2
	store i8 103, i8* %26
	%27 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 3
	store i8 105, i8* %27
	%28 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 4
	store i8 116, i8* %28
	%29 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 5
	store i8 0, i8* %29
	%30 = getelementptr [6 x i8], [6 x i8]* @.str.2, i32 0, i32 0
	%31 = bitcast i8* %30 to i32*
	%32 = call i32 @rt_streq(i32* %23, i32* %31)
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %37

36:
	br label %39

37:
	%38 = load i32, i32* %2
	br label %39

39:
	%40 = phi i32 [ 2, %36 ], [ %38, %37 ]
	store i32 %40, i32* %2
	%41 = load i32*, i32** %1
	%42 = bitcast i32* %41 to i32*
	%43 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 0
	store i8 97, i8* %43
	%44 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 1
	store i8 108, i8* %44
	%45 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 2
	store i8 110, i8* %45
	%46 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 3
	store i8 117, i8* %46
	%47 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 4
	store i8 109, i8* %47
	%48 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 5
	store i8 0, i8* %48
	%49 = getelementptr [6 x i8], [6 x i8]* @.str.3, i32 0, i32 0
	%50 = bitcast i8* %49 to i32*
	%51 = call i32 @rt_streq(i32* %42, i32* %50)
	%52 = icmp ne i32 %51, 0
	%53 = zext i1 %52 to i32
	%54 = icmp ne i32 %53, 0
	br i1 %54, label %55, label %56

55:
	br label %58

56:
	%57 = load i32, i32* %2
	br label %58

58:
	%59 = phi i32 [ 3, %55 ], [ %57, %56 ]
	store i32 %59, i32* %2
	%60 = load i32*, i32** %1
	%61 = bitcast i32* %60 to i32*
	%62 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 0
	store i8 117, i8* %62
	%63 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 1
	store i8 112, i8* %63
	%64 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 2
	store i8 112, i8* %64
	%65 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 3
	store i8 101, i8* %65
	%66 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 4
	store i8 114, i8* %66
	%67 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 5
	store i8 0, i8* %67
	%68 = getelementptr [6 x i8], [6 x i8]* @.str.4, i32 0, i32 0
	%69 = bitcast i8* %68 to i32*
	%70 = call i32 @rt_streq(i32* %61, i32* %69)
	%71 = icmp ne i32 %70, 0
	%72 = zext i1 %71 to i32
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %74, label %75

74:
	br label %77

75:
	%76 = load i32, i32* %2
	br label %77

77:
	%78 = phi i32 [ 4, %74 ], [ %76, %75 ]
	store i32 %78, i32* %2
	%79 = load i32*, i32** %1
	%80 = bitcast i32* %79 to i32*
	%81 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 0
	store i8 108, i8* %81
	%82 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 1
	store i8 111, i8* %82
	%83 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 2
	store i8 119, i8* %83
	%84 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 3
	store i8 101, i8* %84
	%85 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 4
	store i8 114, i8* %85
	%86 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 5
	store i8 0, i8* %86
	%87 = getelementptr [6 x i8], [6 x i8]* @.str.5, i32 0, i32 0
	%88 = bitcast i8* %87 to i32*
	%89 = call i32 @rt_streq(i32* %80, i32* %88)
	%90 = icmp ne i32 %89, 0
	%91 = zext i1 %90 to i32
	%92 = icmp ne i32 %91, 0
	br i1 %92, label %93, label %94

93:
	br label %96

94:
	%95 = load i32, i32* %2
	br label %96

96:
	%97 = phi i32 [ 5, %93 ], [ %95, %94 ]
	store i32 %97, i32* %2
	%98 = load i32*, i32** %1
	%99 = bitcast i32* %98 to i32*
	%100 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 0
	store i8 115, i8* %100
	%101 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 1
	store i8 112, i8* %101
	%102 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 2
	store i8 97, i8* %102
	%103 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 3
	store i8 99, i8* %103
	%104 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 4
	store i8 101, i8* %104
	%105 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 5
	store i8 0, i8* %105
	%106 = getelementptr [6 x i8], [6 x i8]* @.str.6, i32 0, i32 0
	%107 = bitcast i8* %106 to i32*
	%108 = call i32 @rt_streq(i32* %99, i32* %107)
	%109 = icmp ne i32 %108, 0
	%110 = zext i1 %109 to i32
	%111 = icmp ne i32 %110, 0
	br i1 %111, label %112, label %113

112:
	br label %115

113:
	%114 = load i32, i32* %2
	br label %115

115:
	%116 = phi i32 [ 6, %112 ], [ %114, %113 ]
	store i32 %116, i32* %2
	%117 = load i32*, i32** %1
	%118 = bitcast i32* %117 to i32*
	%119 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 0
	store i8 98, i8* %119
	%120 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 1
	store i8 108, i8* %120
	%121 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 2
	store i8 97, i8* %121
	%122 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 3
	store i8 110, i8* %122
	%123 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 4
	store i8 107, i8* %123
	%124 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 5
	store i8 0, i8* %124
	%125 = getelementptr [6 x i8], [6 x i8]* @.str.7, i32 0, i32 0
	%126 = bitcast i8* %125 to i32*
	%127 = call i32 @rt_streq(i32* %118, i32* %126)
	%128 = icmp ne i32 %127, 0
	%129 = zext i1 %128 to i32
	%130 = icmp ne i32 %129, 0
	br i1 %130, label %131, label %132

131:
	br label %134

132:
	%133 = load i32, i32* %2
	br label %134

134:
	%135 = phi i32 [ 7, %131 ], [ %133, %132 ]
	store i32 %135, i32* %2
	%136 = load i32*, i32** %1
	%137 = bitcast i32* %136 to i32*
	%138 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 0
	store i8 112, i8* %138
	%139 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 1
	store i8 117, i8* %139
	%140 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 2
	store i8 110, i8* %140
	%141 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 3
	store i8 99, i8* %141
	%142 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 4
	store i8 116, i8* %142
	%143 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 5
	store i8 0, i8* %143
	%144 = getelementptr [6 x i8], [6 x i8]* @.str.8, i32 0, i32 0
	%145 = bitcast i8* %144 to i32*
	%146 = call i32 @rt_streq(i32* %137, i32* %145)
	%147 = icmp ne i32 %146, 0
	%148 = zext i1 %147 to i32
	%149 = icmp ne i32 %148, 0
	br i1 %149, label %150, label %151

150:
	br label %153

151:
	%152 = load i32, i32* %2
	br label %153

153:
	%154 = phi i32 [ 8, %150 ], [ %152, %151 ]
	store i32 %154, i32* %2
	%155 = load i32*, i32** %1
	%156 = bitcast i32* %155 to i32*
	%157 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 0
	store i8 112, i8* %157
	%158 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 1
	store i8 114, i8* %158
	%159 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 2
	store i8 105, i8* %159
	%160 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 3
	store i8 110, i8* %160
	%161 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 4
	store i8 116, i8* %161
	%162 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 5
	store i8 0, i8* %162
	%163 = getelementptr [6 x i8], [6 x i8]* @.str.9, i32 0, i32 0
	%164 = bitcast i8* %163 to i32*
	%165 = call i32 @rt_streq(i32* %156, i32* %164)
	%166 = icmp ne i32 %165, 0
	%167 = zext i1 %166 to i32
	%168 = icmp ne i32 %167, 0
	br i1 %168, label %169, label %170

169:
	br label %172

170:
	%171 = load i32, i32* %2
	br label %172

172:
	%173 = phi i32 [ 9, %169 ], [ %171, %170 ]
	store i32 %173, i32* %2
	%174 = load i32*, i32** %1
	%175 = bitcast i32* %174 to i32*
	%176 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 0
	store i8 103, i8* %176
	%177 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 1
	store i8 114, i8* %177
	%178 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 2
	store i8 97, i8* %178
	%179 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 3
	store i8 112, i8* %179
	%180 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 4
	store i8 104, i8* %180
	%181 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 5
	store i8 0, i8* %181
	%182 = getelementptr [6 x i8], [6 x i8]* @.str.10, i32 0, i32 0
	%183 = bitcast i8* %182 to i32*
	%184 = call i32 @rt_streq(i32* %175, i32* %183)
	%185 = icmp ne i32 %184, 0
	%186 = zext i1 %185 to i32
	%187 = icmp ne i32 %186, 0
	br i1 %187, label %188, label %189

188:
	br label %191

189:
	%190 = load i32, i32* %2
	br label %191

191:
	%192 = phi i32 [ 10, %188 ], [ %190, %189 ]
	store i32 %192, i32* %2
	%193 = load i32*, i32** %1
	%194 = bitcast i32* %193 to i32*
	%195 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 0
	store i8 99, i8* %195
	%196 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 1
	store i8 110, i8* %196
	%197 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 2
	store i8 116, i8* %197
	%198 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 3
	store i8 114, i8* %198
	%199 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 4
	store i8 108, i8* %199
	%200 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 5
	store i8 0, i8* %200
	%201 = getelementptr [6 x i8], [6 x i8]* @.str.11, i32 0, i32 0
	%202 = bitcast i8* %201 to i32*
	%203 = call i32 @rt_streq(i32* %194, i32* %202)
	%204 = icmp ne i32 %203, 0
	%205 = zext i1 %204 to i32
	%206 = icmp ne i32 %205, 0
	br i1 %206, label %207, label %208

207:
	br label %210

208:
	%209 = load i32, i32* %2
	br label %210

210:
	%211 = phi i32 [ 11, %207 ], [ %209, %208 ]
	store i32 %211, i32* %2
	%212 = load i32*, i32** %1
	%213 = bitcast i32* %212 to i32*
	%214 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 0
	store i8 120, i8* %214
	%215 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 1
	store i8 100, i8* %215
	%216 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 2
	store i8 105, i8* %216
	%217 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 3
	store i8 103, i8* %217
	%218 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 4
	store i8 105, i8* %218
	%219 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 5
	store i8 116, i8* %219
	%220 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 6
	store i8 0, i8* %220
	%221 = getelementptr [7 x i8], [7 x i8]* @.str.12, i32 0, i32 0
	%222 = bitcast i8* %221 to i32*
	%223 = call i32 @rt_streq(i32* %213, i32* %222)
	%224 = icmp ne i32 %223, 0
	%225 = zext i1 %224 to i32
	%226 = icmp ne i32 %225, 0
	br i1 %226, label %227, label %228

227:
	br label %230

228:
	%229 = load i32, i32* %2
	br label %230

230:
	%231 = phi i32 [ 12, %227 ], [ %229, %228 ]
	store i32 %231, i32* %2
	%232 = load i32*, i32** %1
	%233 = bitcast i32* %232 to i32*
	%234 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 0
	store i8 119, i8* %234
	%235 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 1
	store i8 111, i8* %235
	%236 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 2
	store i8 114, i8* %236
	%237 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 3
	store i8 100, i8* %237
	%238 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 4
	store i8 0, i8* %238
	%239 = getelementptr [5 x i8], [5 x i8]* @.str.13, i32 0, i32 0
	%240 = bitcast i8* %239 to i32*
	%241 = call i32 @rt_streq(i32* %233, i32* %240)
	%242 = icmp ne i32 %241, 0
	%243 = zext i1 %242 to i32
	%244 = icmp ne i32 %243, 0
	br i1 %244, label %245, label %246

245:
	br label %248

246:
	%247 = load i32, i32* %2
	br label %248

248:
	%249 = phi i32 [ 13, %245 ], [ %247, %246 ]
	store i32 %249, i32* %2
	%250 = load i32*, i32** %1
	%251 = bitcast i32* %250 to i32*
	%252 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 0
	store i8 97, i8* %252
	%253 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 1
	store i8 115, i8* %253
	%254 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 2
	store i8 99, i8* %254
	%255 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 3
	store i8 105, i8* %255
	%256 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 4
	store i8 105, i8* %256
	%257 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 5
	store i8 0, i8* %257
	%258 = getelementptr [6 x i8], [6 x i8]* @.str.14, i32 0, i32 0
	%259 = bitcast i8* %258 to i32*
	%260 = call i32 @rt_streq(i32* %251, i32* %259)
	%261 = icmp ne i32 %260, 0
	%262 = zext i1 %261 to i32
	%263 = icmp ne i32 %262, 0
	br i1 %263, label %264, label %265

264:
	br label %267

265:
	%266 = load i32, i32* %2
	br label %267

267:
	%268 = phi i32 [ 14, %264 ], [ %266, %265 ]
	store i32 %268, i32* %2
	%269 = load i32, i32* %2
	ret i32 %269

dead17:
	ret i32 0
}

define i32 @rt_class(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	store i32 1, i32* %4
	%5 = alloca i32
	store i32 0, i32* %5
	%6 = alloca i32
	store i32 0, i32* %6
	%7 = alloca i32
	store i32 1, i32* %7
	%8 = alloca i32
	%9 = load i32*, i32** %2
	%10 = getelementptr i8, i32* %9, i32 1
	%11 = load i8, i8* %10
	%12 = sext i8 %11 to i32
	%13 = and i32 %12, 255
	store i32 %13, i32* %8
	%14 = load i32, i32* %8
	%15 = icmp eq i32 %14, 33
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %26, label %20

20:
	%21 = load i32, i32* %8
	%22 = icmp eq i32 %21, 94
	%23 = zext i1 %22 to i32
	%24 = icmp ne i32 %23, 0
	%25 = zext i1 %24 to i32
	br label %26

26:
	%27 = phi i32 [ %18, %entry ], [ %25, %20 ]
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %30

29:
	store i32 1, i32* %5
	store i32 2, i32* %4
	br label %30

30:
	br label %31

31:
	%32 = icmp ne i32 1, 0
	br i1 %32, label %33, label %45

33:
	%34 = alloca i32
	%35 = load i32, i32* %4
	%36 = load i32*, i32** %2
	%37 = getelementptr i8, i32* %36, i32 %35
	%38 = load i8, i8* %37
	%39 = sext i8 %38 to i32
	%40 = and i32 %39, 255
	store i32 %40, i32* %34
	%41 = load i32, i32* %34
	%42 = icmp eq i32 %41, 0
	%43 = zext i1 %42 to i32
	%44 = icmp ne i32 %43, 0
	br i1 %44, label %49, label %50

45:
	%46 = alloca i32
	%47 = load i32, i32* %5
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %292, label %295

49:
	ret i32 -1

50:
	%51 = load i32, i32* %34
	%52 = icmp eq i32 %51, 93
	%53 = zext i1 %52 to i32
	%54 = icmp ne i32 %53, 0
	%55 = zext i1 %54 to i32
	%56 = icmp ne i32 %55, 0
	br i1 %56, label %57, label %63

dead18:
	br label %50

57:
	%58 = load i32, i32* %7
	%59 = icmp eq i32 %58, 0
	%60 = zext i1 %59 to i32
	%61 = icmp ne i32 %60, 0
	%62 = zext i1 %61 to i32
	br label %63

63:
	%64 = phi i32 [ %55, %50 ], [ %62, %57 ]
	%65 = icmp ne i32 %64, 0
	br i1 %65, label %66, label %67

66:
	br label %45

67:
	store i32 0, i32* %7
	%68 = load i32, i32* %34
	%69 = icmp eq i32 %68, 91
	%70 = zext i1 %69 to i32
	%71 = icmp ne i32 %70, 0
	%72 = zext i1 %71 to i32
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %74, label %86

dead19:
	br label %67

74:
	%75 = load i32, i32* %4
	%76 = add i32 %75, 1
	%77 = load i32*, i32** %2
	%78 = getelementptr i8, i32* %77, i32 %76
	%79 = load i8, i8* %78
	%80 = sext i8 %79 to i32
	%81 = and i32 %80, 255
	%82 = icmp eq i32 %81, 58
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	%85 = zext i1 %84 to i32
	br label %86

86:
	%87 = phi i32 [ %72, %67 ], [ %85, %74 ]
	%88 = icmp ne i32 %87, 0
	br i1 %88, label %89, label %92

89:
	%90 = alloca i32
	store i32 0, i32* %90
	%91 = alloca i32
	store i32 0, i32* %91
	br label %106

92:
	%93 = alloca i32
	%94 = load i32, i32* %4
	%95 = add i32 %94, 1
	%96 = load i32*, i32** %2
	%97 = getelementptr i8, i32* %96, i32 %95
	%98 = load i8, i8* %97
	%99 = sext i8 %98 to i32
	%100 = and i32 %99, 255
	store i32 %100, i32* %93
	%101 = alloca i32
	store i32 0, i32* %101
	%102 = load i32, i32* %93
	%103 = icmp eq i32 %102, 45
	%104 = zext i1 %103 to i32
	%105 = icmp ne i32 %104, 0
	br i1 %105, label %226, label %234

106:
	%107 = icmp ne i32 1, 0
	br i1 %107, label %108, label %123

108:
	%109 = alloca i32
	%110 = load i32, i32* %4
	%111 = add i32 %110, 2
	%112 = load i32, i32* %90
	%113 = add i32 %111, %112
	%114 = load i32*, i32** %2
	%115 = getelementptr i8, i32* %114, i32 %113
	%116 = load i8, i8* %115
	%117 = sext i8 %116 to i32
	%118 = and i32 %117, 255
	store i32 %118, i32* %109
	%119 = load i32, i32* %109
	%120 = icmp eq i32 %119, 0
	%121 = zext i1 %120 to i32
	%122 = icmp ne i32 %121, 0
	br i1 %122, label %128, label %129

123:
	%124 = load i32, i32* %91
	%125 = icmp ne i32 %124, 0
	%126 = zext i1 %125 to i32
	%127 = icmp ne i32 %126, 0
	br i1 %127, label %157, label %165

128:
	br label %123

129:
	%130 = load i32, i32* %109
	%131 = icmp eq i32 %130, 58
	%132 = zext i1 %131 to i32
	%133 = icmp ne i32 %132, 0
	%134 = zext i1 %133 to i32
	%135 = icmp ne i32 %134, 0
	br i1 %135, label %136, label %150

dead20:
	br label %129

136:
	%137 = load i32, i32* %4
	%138 = add i32 %137, 3
	%139 = load i32, i32* %90
	%140 = add i32 %138, %139
	%141 = load i32*, i32** %2
	%142 = getelementptr i8, i32* %141, i32 %140
	%143 = load i8, i8* %142
	%144 = sext i8 %143 to i32
	%145 = and i32 %144, 255
	%146 = icmp eq i32 %145, 93
	%147 = zext i1 %146 to i32
	%148 = icmp ne i32 %147, 0
	%149 = zext i1 %148 to i32
	br label %150

150:
	%151 = phi i32 [ %134, %129 ], [ %149, %136 ]
	%152 = icmp ne i32 %151, 0
	br i1 %152, label %153, label %154

153:
	store i32 1, i32* %91
	br label %123

154:
	%155 = load i32, i32* %90
	%156 = add i32 %155, 1
	store i32 %156, i32* %90
	br label %106

dead21:
	br label %154

157:
	%158 = alloca i32*
	%159 = load i32, i32* %90
	%160 = add i32 %159, 1
	%161 = call i32* @rt_bump(i32 %160)
	%162 = bitcast i32* %161 to i32*
	store i32* %162, i32** %158
	%163 = alloca i32
	store i32 0, i32* %163
	%164 = alloca i32
	br label %166

165:
	br label %92

166:
	%167 = load i32, i32* %163
	%168 = load i32, i32* %90
	%169 = icmp slt i32 %167, %168
	%170 = zext i1 %169 to i32
	%171 = icmp ne i32 %170, 0
	br i1 %171, label %172, label %191

172:
	%173 = load i32, i32* %163
	%174 = load i32*, i32** %158
	%175 = getelementptr i8, i32* %174, i32 %173
	%176 = load i32, i32* %4
	%177 = add i32 %176, 2
	%178 = load i32, i32* %163
	%179 = add i32 %177, %178
	%180 = load i32*, i32** %2
	%181 = getelementptr i8, i32* %180, i32 %179
	%182 = load i8, i8* %181
	%183 = sext i8 %182 to i32
	%184 = shl i32 %183, 24
	%185 = ashr i32 %184, 24
	%186 = shl i32 %185, 24
	%187 = ashr i32 %186, 24
	%188 = trunc i32 %187 to i8
	store i8 %188, i8* %175
	%189 = load i32, i32* %163
	%190 = add i32 %189, 1
	store i32 %190, i32* %163
	br label %166

191:
	%192 = load i32, i32* %90
	%193 = load i32*, i32** %158
	%194 = getelementptr i8, i32* %193, i32 %192
	%195 = shl i32 0, 24
	%196 = ashr i32 %195, 24
	%197 = shl i32 %196, 24
	%198 = ashr i32 %197, 24
	%199 = trunc i32 %198 to i8
	store i8 %199, i8* %194
	%200 = load i32*, i32** %158
	%201 = bitcast i32* %200 to i32*
	%202 = call i32 @rt_clsid(i32* %201)
	store i32 %202, i32* %164
	%203 = load i32, i32* %164
	%204 = icmp ne i32 %203, 0
	%205 = zext i1 %204 to i32
	%206 = icmp ne i32 %205, 0
	%207 = zext i1 %206 to i32
	%208 = icmp ne i32 %207, 0
	br i1 %208, label %209, label %217

209:
	%210 = load i32, i32* %164
	%211 = load i32, i32* %3
	%212 = call i32 @re_ctype(i32 %210, i32 %211)
	%213 = icmp ne i32 %212, 0
	%214 = zext i1 %213 to i32
	%215 = icmp ne i32 %214, 0
	%216 = zext i1 %215 to i32
	br label %217

217:
	%218 = phi i32 [ %207, %191 ], [ %216, %209 ]
	%219 = icmp ne i32 %218, 0
	br i1 %219, label %220, label %221

220:
	store i32 1, i32* %6
	br label %221

221:
	%222 = load i32, i32* %4
	%223 = load i32, i32* %90
	%224 = add i32 %222, %223
	%225 = add i32 %224, 4
	store i32 %225, i32* %4
	br label %31

dead22:
	br label %165

226:
	%227 = load i32, i32* %4
	%228 = add i32 %227, 2
	%229 = load i32*, i32** %2
	%230 = getelementptr i8, i32* %229, i32 %228
	%231 = load i8, i8* %230
	%232 = sext i8 %231 to i32
	%233 = and i32 %232, 255
	store i32 %233, i32* %101
	br label %234

234:
	%235 = load i32, i32* %93
	%236 = icmp eq i32 %235, 45
	%237 = zext i1 %236 to i32
	%238 = icmp ne i32 %237, 0
	%239 = zext i1 %238 to i32
	%240 = icmp ne i32 %239, 0
	br i1 %240, label %241, label %247

241:
	%242 = load i32, i32* %101
	%243 = icmp ne i32 %242, 0
	%244 = zext i1 %243 to i32
	%245 = icmp ne i32 %244, 0
	%246 = zext i1 %245 to i32
	br label %247

247:
	%248 = phi i32 [ %239, %234 ], [ %246, %241 ]
	%249 = icmp ne i32 %248, 0
	br i1 %249, label %250, label %256

250:
	%251 = load i32, i32* %101
	%252 = icmp ne i32 %251, 93
	%253 = zext i1 %252 to i32
	%254 = icmp ne i32 %253, 0
	%255 = zext i1 %254 to i32
	br label %256

256:
	%257 = phi i32 [ %248, %247 ], [ %255, %250 ]
	%258 = icmp ne i32 %257, 0
	br i1 %258, label %259, label %268

259:
	%260 = load i32, i32* %3
	%261 = load i32, i32* %34
	%262 = icmp sge i32 %260, %261
	%263 = zext i1 %262 to i32
	%264 = icmp ne i32 %263, 0
	%265 = zext i1 %264 to i32
	%266 = icmp ne i32 %265, 0
	br i1 %266, label %274, label %281

267:
	br label %31

268:
	%269 = load i32, i32* %3
	%270 = load i32, i32* %34
	%271 = icmp eq i32 %269, %270
	%272 = zext i1 %271 to i32
	%273 = icmp ne i32 %272, 0
	br i1 %273, label %288, label %289

274:
	%275 = load i32, i32* %3
	%276 = load i32, i32* %101
	%277 = icmp sle i32 %275, %276
	%278 = zext i1 %277 to i32
	%279 = icmp ne i32 %278, 0
	%280 = zext i1 %279 to i32
	br label %281

281:
	%282 = phi i32 [ %265, %259 ], [ %280, %274 ]
	%283 = icmp ne i32 %282, 0
	br i1 %283, label %284, label %285

284:
	store i32 1, i32* %6
	br label %285

285:
	%286 = load i32, i32* %4
	%287 = add i32 %286, 3
	store i32 %287, i32* %4
	br label %267

288:
	store i32 1, i32* %6
	br label %289

289:
	%290 = load i32, i32* %4
	%291 = add i32 %290, 1
	store i32 %291, i32* %4
	br label %267

292:
	%293 = load i32, i32* %6
	%294 = sub i32 1, %293
	br label %297

295:
	%296 = load i32, i32* %6
	br label %297

297:
	%298 = phi i32 [ %294, %292 ], [ %296, %295 ]
	store i32 %298, i32* %46
	%299 = load i32, i32* %4
	%300 = add i32 %299, 1
	%301 = mul i32 %300, 2
	%302 = load i32, i32* %46
	%303 = or i32 %301, %302
	ret i32 %303

dead23:
	ret i32 0
}

define i32 @re_ctype(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = load i32, i32* %3
	%12 = icmp sge i32 %11, 65
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	%15 = zext i1 %14 to i32
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %23

17:
	%18 = load i32, i32* %3
	%19 = icmp sle i32 %18, 90
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	br label %23

23:
	%24 = phi i32 [ %15, %entry ], [ %22, %17 ]
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %27

26:
	br label %28

27:
	br label %28

28:
	%29 = phi i32 [ 1, %26 ], [ 0, %27 ]
	store i32 %29, i32* %4
	%30 = load i32, i32* %3
	%31 = icmp sge i32 %30, 97
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %42

36:
	%37 = load i32, i32* %3
	%38 = icmp sle i32 %37, 122
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	%41 = zext i1 %40 to i32
	br label %42

42:
	%43 = phi i32 [ %34, %28 ], [ %41, %36 ]
	%44 = icmp ne i32 %43, 0
	br i1 %44, label %45, label %46

45:
	br label %47

46:
	br label %47

47:
	%48 = phi i32 [ 1, %45 ], [ 0, %46 ]
	store i32 %48, i32* %5
	%49 = load i32, i32* %3
	%50 = icmp sge i32 %49, 48
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	%53 = zext i1 %52 to i32
	%54 = icmp ne i32 %53, 0
	br i1 %54, label %55, label %61

55:
	%56 = load i32, i32* %3
	%57 = icmp sle i32 %56, 57
	%58 = zext i1 %57 to i32
	%59 = icmp ne i32 %58, 0
	%60 = zext i1 %59 to i32
	br label %61

61:
	%62 = phi i32 [ %53, %47 ], [ %60, %55 ]
	%63 = icmp ne i32 %62, 0
	br i1 %63, label %64, label %65

64:
	br label %66

65:
	br label %66

66:
	%67 = phi i32 [ 1, %64 ], [ 0, %65 ]
	store i32 %67, i32* %6
	%68 = load i32, i32* %4
	%69 = load i32, i32* %5
	%70 = or i32 %68, %69
	store i32 %70, i32* %7
	%71 = load i32, i32* %7
	%72 = load i32, i32* %6
	%73 = or i32 %71, %72
	store i32 %73, i32* %8
	%74 = load i32, i32* %3
	%75 = icmp sge i32 %74, 33
	%76 = zext i1 %75 to i32
	%77 = icmp ne i32 %76, 0
	%78 = zext i1 %77 to i32
	%79 = icmp ne i32 %78, 0
	br i1 %79, label %80, label %86

80:
	%81 = load i32, i32* %3
	%82 = icmp sle i32 %81, 126
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	%85 = zext i1 %84 to i32
	br label %86

86:
	%87 = phi i32 [ %78, %66 ], [ %85, %80 ]
	%88 = icmp ne i32 %87, 0
	br i1 %88, label %89, label %90

89:
	br label %91

90:
	br label %91

91:
	%92 = phi i32 [ 1, %89 ], [ 0, %90 ]
	store i32 %92, i32* %9
	store i32 0, i32* %10
	%93 = load i32, i32* %2
	%94 = icmp eq i32 %93, 1
	%95 = zext i1 %94 to i32
	%96 = icmp ne i32 %95, 0
	br i1 %96, label %97, label %99

97:
	%98 = load i32, i32* %7
	br label %101

99:
	%100 = load i32, i32* %10
	br label %101

101:
	%102 = phi i32 [ %98, %97 ], [ %100, %99 ]
	store i32 %102, i32* %10
	%103 = load i32, i32* %2
	%104 = icmp eq i32 %103, 2
	%105 = zext i1 %104 to i32
	%106 = icmp ne i32 %105, 0
	br i1 %106, label %107, label %109

107:
	%108 = load i32, i32* %6
	br label %111

109:
	%110 = load i32, i32* %10
	br label %111

111:
	%112 = phi i32 [ %108, %107 ], [ %110, %109 ]
	store i32 %112, i32* %10
	%113 = load i32, i32* %2
	%114 = icmp eq i32 %113, 3
	%115 = zext i1 %114 to i32
	%116 = icmp ne i32 %115, 0
	br i1 %116, label %117, label %119

117:
	%118 = load i32, i32* %8
	br label %121

119:
	%120 = load i32, i32* %10
	br label %121

121:
	%122 = phi i32 [ %118, %117 ], [ %120, %119 ]
	store i32 %122, i32* %10
	%123 = load i32, i32* %2
	%124 = icmp eq i32 %123, 4
	%125 = zext i1 %124 to i32
	%126 = icmp ne i32 %125, 0
	br i1 %126, label %127, label %129

127:
	%128 = load i32, i32* %4
	br label %131

129:
	%130 = load i32, i32* %10
	br label %131

131:
	%132 = phi i32 [ %128, %127 ], [ %130, %129 ]
	store i32 %132, i32* %10
	%133 = load i32, i32* %2
	%134 = icmp eq i32 %133, 5
	%135 = zext i1 %134 to i32
	%136 = icmp ne i32 %135, 0
	br i1 %136, label %137, label %139

137:
	%138 = load i32, i32* %5
	br label %141

139:
	%140 = load i32, i32* %10
	br label %141

141:
	%142 = phi i32 [ %138, %137 ], [ %140, %139 ]
	store i32 %142, i32* %10
	%143 = load i32, i32* %2
	%144 = icmp eq i32 %143, 6
	%145 = zext i1 %144 to i32
	%146 = icmp ne i32 %145, 0
	br i1 %146, label %147, label %152

147:
	%148 = load i32, i32* %3
	%149 = icmp eq i32 %148, 32
	%150 = zext i1 %149 to i32
	%151 = icmp ne i32 %150, 0
	br i1 %151, label %160, label %161

152:
	%153 = load i32, i32* %10
	br label %154

154:
	%155 = phi i32 [ %183, %181 ], [ %153, %152 ]
	store i32 %155, i32* %10
	%156 = load i32, i32* %2
	%157 = icmp eq i32 %156, 7
	%158 = zext i1 %157 to i32
	%159 = icmp ne i32 %158, 0
	br i1 %159, label %184, label %189

160:
	br label %162

161:
	br label %162

162:
	%163 = phi i32 [ 1, %160 ], [ 0, %161 ]
	%164 = load i32, i32* %3
	%165 = icmp sge i32 %164, 9
	%166 = zext i1 %165 to i32
	%167 = icmp ne i32 %166, 0
	%168 = zext i1 %167 to i32
	%169 = icmp ne i32 %168, 0
	br i1 %169, label %170, label %176

170:
	%171 = load i32, i32* %3
	%172 = icmp sle i32 %171, 13
	%173 = zext i1 %172 to i32
	%174 = icmp ne i32 %173, 0
	%175 = zext i1 %174 to i32
	br label %176

176:
	%177 = phi i32 [ %168, %162 ], [ %175, %170 ]
	%178 = icmp ne i32 %177, 0
	br i1 %178, label %179, label %180

179:
	br label %181

180:
	br label %181

181:
	%182 = phi i32 [ 1, %179 ], [ 0, %180 ]
	%183 = or i32 %163, %182
	br label %154

184:
	%185 = load i32, i32* %3
	%186 = icmp eq i32 %185, 32
	%187 = zext i1 %186 to i32
	%188 = icmp ne i32 %187, 0
	br i1 %188, label %197, label %198

189:
	%190 = load i32, i32* %10
	br label %191

191:
	%192 = phi i32 [ %209, %207 ], [ %190, %189 ]
	store i32 %192, i32* %10
	%193 = load i32, i32* %2
	%194 = icmp eq i32 %193, 8
	%195 = zext i1 %194 to i32
	%196 = icmp ne i32 %195, 0
	br i1 %196, label %210, label %215

197:
	br label %199

198:
	br label %199

199:
	%200 = phi i32 [ 1, %197 ], [ 0, %198 ]
	%201 = load i32, i32* %3
	%202 = icmp eq i32 %201, 9
	%203 = zext i1 %202 to i32
	%204 = icmp ne i32 %203, 0
	br i1 %204, label %205, label %206

205:
	br label %207

206:
	br label %207

207:
	%208 = phi i32 [ 1, %205 ], [ 0, %206 ]
	%209 = or i32 %200, %208
	br label %191

210:
	%211 = load i32, i32* %9
	%212 = load i32, i32* %8
	%213 = sub i32 1, %212
	%214 = and i32 %211, %213
	br label %217

215:
	%216 = load i32, i32* %10
	br label %217

217:
	%218 = phi i32 [ %214, %210 ], [ %216, %215 ]
	store i32 %218, i32* %10
	%219 = load i32, i32* %2
	%220 = icmp eq i32 %219, 9
	%221 = zext i1 %220 to i32
	%222 = icmp ne i32 %221, 0
	br i1 %222, label %223, label %230

223:
	%224 = load i32, i32* %3
	%225 = icmp sge i32 %224, 32
	%226 = zext i1 %225 to i32
	%227 = icmp ne i32 %226, 0
	%228 = zext i1 %227 to i32
	%229 = icmp ne i32 %228, 0
	br i1 %229, label %238, label %244

230:
	%231 = load i32, i32* %10
	br label %232

232:
	%233 = phi i32 [ %250, %249 ], [ %231, %230 ]
	store i32 %233, i32* %10
	%234 = load i32, i32* %2
	%235 = icmp eq i32 %234, 10
	%236 = zext i1 %235 to i32
	%237 = icmp ne i32 %236, 0
	br i1 %237, label %251, label %253

238:
	%239 = load i32, i32* %3
	%240 = icmp sle i32 %239, 126
	%241 = zext i1 %240 to i32
	%242 = icmp ne i32 %241, 0
	%243 = zext i1 %242 to i32
	br label %244

244:
	%245 = phi i32 [ %228, %223 ], [ %243, %238 ]
	%246 = icmp ne i32 %245, 0
	br i1 %246, label %247, label %248

247:
	br label %249

248:
	br label %249

249:
	%250 = phi i32 [ 1, %247 ], [ 0, %248 ]
	br label %232

251:
	%252 = load i32, i32* %9
	br label %255

253:
	%254 = load i32, i32* %10
	br label %255

255:
	%256 = phi i32 [ %252, %251 ], [ %254, %253 ]
	store i32 %256, i32* %10
	%257 = load i32, i32* %2
	%258 = icmp eq i32 %257, 11
	%259 = zext i1 %258 to i32
	%260 = icmp ne i32 %259, 0
	br i1 %260, label %261, label %268

261:
	%262 = load i32, i32* %3
	%263 = icmp sge i32 %262, 0
	%264 = zext i1 %263 to i32
	%265 = icmp ne i32 %264, 0
	%266 = zext i1 %265 to i32
	%267 = icmp ne i32 %266, 0
	br i1 %267, label %276, label %282

268:
	%269 = load i32, i32* %10
	br label %270

270:
	%271 = phi i32 [ %297, %295 ], [ %269, %268 ]
	store i32 %271, i32* %10
	%272 = load i32, i32* %2
	%273 = icmp eq i32 %272, 12
	%274 = zext i1 %273 to i32
	%275 = icmp ne i32 %274, 0
	br i1 %275, label %298, label %306

276:
	%277 = load i32, i32* %3
	%278 = icmp sle i32 %277, 31
	%279 = zext i1 %278 to i32
	%280 = icmp ne i32 %279, 0
	%281 = zext i1 %280 to i32
	br label %282

282:
	%283 = phi i32 [ %266, %261 ], [ %281, %276 ]
	%284 = icmp ne i32 %283, 0
	br i1 %284, label %285, label %286

285:
	br label %287

286:
	br label %287

287:
	%288 = phi i32 [ 1, %285 ], [ 0, %286 ]
	%289 = load i32, i32* %3
	%290 = icmp eq i32 %289, 127
	%291 = zext i1 %290 to i32
	%292 = icmp ne i32 %291, 0
	br i1 %292, label %293, label %294

293:
	br label %295

294:
	br label %295

295:
	%296 = phi i32 [ 1, %293 ], [ 0, %294 ]
	%297 = or i32 %288, %296
	br label %270

298:
	%299 = load i32, i32* %6
	%300 = load i32, i32* %3
	%301 = icmp sge i32 %300, 97
	%302 = zext i1 %301 to i32
	%303 = icmp ne i32 %302, 0
	%304 = zext i1 %303 to i32
	%305 = icmp ne i32 %304, 0
	br i1 %305, label %314, label %320

306:
	%307 = load i32, i32* %10
	br label %308

308:
	%309 = phi i32 [ %347, %344 ], [ %307, %306 ]
	store i32 %309, i32* %10
	%310 = load i32, i32* %2
	%311 = icmp eq i32 %310, 13
	%312 = zext i1 %311 to i32
	%313 = icmp ne i32 %312, 0
	br i1 %313, label %348, label %354

314:
	%315 = load i32, i32* %3
	%316 = icmp sle i32 %315, 102
	%317 = zext i1 %316 to i32
	%318 = icmp ne i32 %317, 0
	%319 = zext i1 %318 to i32
	br label %320

320:
	%321 = phi i32 [ %304, %298 ], [ %319, %314 ]
	%322 = icmp ne i32 %321, 0
	br i1 %322, label %323, label %324

323:
	br label %325

324:
	br label %325

325:
	%326 = phi i32 [ 1, %323 ], [ 0, %324 ]
	%327 = load i32, i32* %3
	%328 = icmp sge i32 %327, 65
	%329 = zext i1 %328 to i32
	%330 = icmp ne i32 %329, 0
	%331 = zext i1 %330 to i32
	%332 = icmp ne i32 %331, 0
	br i1 %332, label %333, label %339

333:
	%334 = load i32, i32* %3
	%335 = icmp sle i32 %334, 70
	%336 = zext i1 %335 to i32
	%337 = icmp ne i32 %336, 0
	%338 = zext i1 %337 to i32
	br label %339

339:
	%340 = phi i32 [ %331, %325 ], [ %338, %333 ]
	%341 = icmp ne i32 %340, 0
	br i1 %341, label %342, label %343

342:
	br label %344

343:
	br label %344

344:
	%345 = phi i32 [ 1, %342 ], [ 0, %343 ]
	%346 = or i32 %326, %345
	%347 = or i32 %299, %346
	br label %308

348:
	%349 = load i32, i32* %8
	%350 = load i32, i32* %3
	%351 = icmp eq i32 %350, 95
	%352 = zext i1 %351 to i32
	%353 = icmp ne i32 %352, 0
	br i1 %353, label %362, label %363

354:
	%355 = load i32, i32* %10
	br label %356

356:
	%357 = phi i32 [ %366, %364 ], [ %355, %354 ]
	store i32 %357, i32* %10
	%358 = load i32, i32* %2
	%359 = icmp eq i32 %358, 14
	%360 = zext i1 %359 to i32
	%361 = icmp ne i32 %360, 0
	br i1 %361, label %367, label %374

362:
	br label %364

363:
	br label %364

364:
	%365 = phi i32 [ 1, %362 ], [ 0, %363 ]
	%366 = or i32 %349, %365
	br label %356

367:
	%368 = load i32, i32* %3
	%369 = icmp sge i32 %368, 0
	%370 = zext i1 %369 to i32
	%371 = icmp ne i32 %370, 0
	%372 = zext i1 %371 to i32
	%373 = icmp ne i32 %372, 0
	br i1 %373, label %379, label %385

374:
	%375 = load i32, i32* %10
	br label %376

376:
	%377 = phi i32 [ %391, %390 ], [ %375, %374 ]
	store i32 %377, i32* %10
	%378 = load i32, i32* %10
	ret i32 %378

379:
	%380 = load i32, i32* %3
	%381 = icmp sle i32 %380, 127
	%382 = zext i1 %381 to i32
	%383 = icmp ne i32 %382, 0
	%384 = zext i1 %383 to i32
	br label %385

385:
	%386 = phi i32 [ %372, %367 ], [ %384, %379 ]
	%387 = icmp ne i32 %386, 0
	br i1 %387, label %388, label %389

388:
	br label %390

389:
	br label %390

390:
	%391 = phi i32 [ 1, %388 ], [ 0, %389 ]
	br label %376

dead159:
	ret i32 0
}

define i32* @rt_substr(i32* %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = alloca i32*
	%12 = load i32*, i32** %3
	%13 = bitcast i32* %12 to i32*
	%14 = call i32 @rt_strlen(i32* %13)
	store i32 %14, i32* %6
	%15 = load i32, i32* %4
	%16 = icmp slt i32 %15, 0
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %19, label %20

19:
	br label %22

20:
	%21 = load i32, i32* %4
	br label %22

22:
	%23 = phi i32 [ 0, %19 ], [ %21, %20 ]
	store i32 %23, i32* %7
	%24 = load i32, i32* %7
	%25 = load i32, i32* %6
	%26 = icmp sgt i32 %24, %25
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %31

29:
	%30 = load i32, i32* %6
	br label %33

31:
	%32 = load i32, i32* %7
	br label %33

33:
	%34 = phi i32 [ %30, %29 ], [ %32, %31 ]
	store i32 %34, i32* %7
	%35 = load i32, i32* %6
	%36 = load i32, i32* %7
	%37 = sub i32 %35, %36
	store i32 %37, i32* %8
	%38 = load i32, i32* %5
	%39 = icmp slt i32 %38, 0
	%40 = zext i1 %39 to i32
	%41 = icmp ne i32 %40, 0
	br i1 %41, label %42, label %44

42:
	%43 = load i32, i32* %8
	br label %46

44:
	%45 = load i32, i32* %5
	br label %46

46:
	%47 = phi i32 [ %43, %42 ], [ %45, %44 ]
	store i32 %47, i32* %9
	%48 = load i32, i32* %9
	%49 = load i32, i32* %8
	%50 = icmp sgt i32 %48, %49
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %53, label %55

53:
	%54 = load i32, i32* %8
	br label %57

55:
	%56 = load i32, i32* %9
	br label %57

57:
	%58 = phi i32 [ %54, %53 ], [ %56, %55 ]
	store i32 %58, i32* %9
	%59 = load i32, i32* %9
	%60 = add i32 %59, 1
	%61 = call i32* @rt_bump(i32 %60)
	%62 = bitcast i32* %61 to i32*
	store i32* %62, i32** %11
	store i32 0, i32* %10
	br label %63

63:
	%64 = load i32, i32* %10
	%65 = load i32, i32* %9
	%66 = icmp slt i32 %64, %65
	%67 = zext i1 %66 to i32
	%68 = icmp ne i32 %67, 0
	br i1 %68, label %69, label %87

69:
	%70 = load i32, i32* %10
	%71 = load i32*, i32** %11
	%72 = getelementptr i8, i32* %71, i32 %70
	%73 = load i32, i32* %7
	%74 = load i32, i32* %10
	%75 = add i32 %73, %74
	%76 = load i32*, i32** %3
	%77 = getelementptr i8, i32* %76, i32 %75
	%78 = load i8, i8* %77
	%79 = sext i8 %78 to i32
	%80 = shl i32 %79, 24
	%81 = ashr i32 %80, 24
	%82 = shl i32 %81, 24
	%83 = ashr i32 %82, 24
	%84 = trunc i32 %83 to i8
	store i8 %84, i8* %72
	%85 = load i32, i32* %10
	%86 = add i32 %85, 1
	store i32 %86, i32* %10
	br label %63

87:
	%88 = load i32, i32* %9
	%89 = load i32*, i32** %11
	%90 = getelementptr i8, i32* %89, i32 %88
	%91 = shl i32 0, 24
	%92 = ashr i32 %91, 24
	%93 = shl i32 %92, 24
	%94 = ashr i32 %93, 24
	%95 = trunc i32 %94 to i8
	store i8 %95, i8* %90
	%96 = load i32*, i32** %11
	%97 = bitcast i32* %96 to i32*
	ret i32* %97

dead24:
	ret i32* null
}

define i32* @rt_csubstr(i32* %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = load i32*, i32** %3
	%10 = bitcast i32* %9 to i32*
	%11 = load i32, i32* %4
	%12 = icmp slt i32 %11, 0
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	br i1 %14, label %15, label %16

15:
	br label %18

16:
	%17 = load i32, i32* %4
	br label %18

18:
	%19 = phi i32 [ 0, %15 ], [ %17, %16 ]
	%20 = call i32 @rt_charoff(i32* %10, i32 %19)
	store i32 %20, i32* %6
	%21 = load i32*, i32** %3
	%22 = bitcast i32* %21 to i32*
	%23 = load i32, i32* %4
	%24 = load i32, i32* %5
	%25 = icmp slt i32 %24, 0
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	br i1 %27, label %28, label %29

28:
	br label %31

29:
	%30 = load i32, i32* %5
	br label %31

31:
	%32 = phi i32 [ 0, %28 ], [ %30, %29 ]
	%33 = add i32 %23, %32
	%34 = call i32 @rt_charoff(i32* %22, i32 %33)
	store i32 %34, i32* %7
	%35 = load i32, i32* %5
	%36 = icmp slt i32 %35, 0
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %39, label %40

39:
	br label %44

40:
	%41 = load i32, i32* %7
	%42 = load i32, i32* %6
	%43 = sub i32 %41, %42
	br label %44

44:
	%45 = phi i32 [ -1, %39 ], [ %43, %40 ]
	store i32 %45, i32* %8
	%46 = load i32, i32* %8
	%47 = icmp slt i32 %46, 0
	%48 = zext i1 %47 to i32
	%49 = icmp ne i32 %48, 0
	br i1 %49, label %50, label %51

50:
	br label %53

51:
	%52 = load i32, i32* %8
	br label %53

53:
	%54 = phi i32 [ -1, %50 ], [ %52, %51 ]
	store i32 %54, i32* %8
	%55 = load i32*, i32** %3
	%56 = bitcast i32* %55 to i32*
	%57 = load i32, i32* %6
	%58 = load i32, i32* %8
	%59 = call i32* @rt_substr(i32* %56, i32 %57, i32 %58)
	%60 = bitcast i32* %59 to i32*
	ret i32* %60

dead25:
	ret i32* null
}

define i32 @rt_egclose(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	store i32 0, i32* %2
	store i32 0, i32* %3
	br label %5

5:
	%6 = icmp ne i32 1, 0
	br i1 %6, label %7, label %18

7:
	%8 = load i32, i32* %2
	%9 = load i32*, i32** %1
	%10 = getelementptr i8, i32* %9, i32 %8
	%11 = load i8, i8* %10
	%12 = sext i8 %11 to i32
	%13 = and i32 %12, 255
	store i32 %13, i32* %4
	%14 = load i32, i32* %4
	%15 = icmp eq i32 %14, 0
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %19, label %20

18:
	ret i32 0

19:
	ret i32 -1

20:
	%21 = load i32, i32* %4
	%22 = icmp eq i32 %21, 92
	%23 = zext i1 %22 to i32
	%24 = icmp ne i32 %23, 0
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %27, label %39

dead26:
	br label %20

27:
	%28 = load i32, i32* %2
	%29 = add i32 %28, 1
	%30 = load i32*, i32** %1
	%31 = getelementptr i8, i32* %30, i32 %29
	%32 = load i8, i8* %31
	%33 = sext i8 %32 to i32
	%34 = and i32 %33, 255
	%35 = icmp ne i32 %34, 0
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	%38 = zext i1 %37 to i32
	br label %39

39:
	%40 = phi i32 [ %25, %20 ], [ %38, %27 ]
	%41 = icmp ne i32 %40, 0
	br i1 %41, label %42, label %45

42:
	%43 = load i32, i32* %2
	%44 = add i32 %43, 2
	store i32 %44, i32* %2
	br label %5

45:
	%46 = load i32, i32* %4
	%47 = icmp eq i32 %46, 40
	%48 = zext i1 %47 to i32
	%49 = icmp ne i32 %48, 0
	br i1 %49, label %50, label %56

dead27:
	br label %45

50:
	%51 = load i32, i32* %3
	%52 = add i32 %51, 1
	store i32 %52, i32* %3
	br label %53

53:
	%54 = load i32, i32* %2
	%55 = add i32 %54, 1
	store i32 %55, i32* %2
	br label %5

56:
	%57 = load i32, i32* %4
	%58 = icmp eq i32 %57, 41
	%59 = zext i1 %58 to i32
	%60 = icmp ne i32 %59, 0
	br i1 %60, label %61, label %68

61:
	%62 = load i32, i32* %3
	%63 = sub i32 %62, 1
	store i32 %63, i32* %3
	%64 = load i32, i32* %3
	%65 = icmp eq i32 %64, 0
	%66 = zext i1 %65 to i32
	%67 = icmp ne i32 %66, 0
	br i1 %67, label %69, label %71

68:
	br label %53

69:
	%70 = load i32, i32* %2
	ret i32 %70

71:
	br label %68

dead28:
	br label %71
}

define i32 @rt_egalt(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = load i32, i32* %3
	store i32 %7, i32* %4
	store i32 0, i32* %5
	br label %8

8:
	%9 = icmp ne i32 1, 0
	br i1 %9, label %10, label %21

10:
	%11 = load i32, i32* %4
	%12 = load i32*, i32** %2
	%13 = getelementptr i8, i32* %12, i32 %11
	%14 = load i8, i8* %13
	%15 = sext i8 %14 to i32
	%16 = and i32 %15, 255
	store i32 %16, i32* %6
	%17 = load i32, i32* %6
	%18 = icmp eq i32 %17, 0
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %22, label %24

21:
	ret i32 0

22:
	%23 = load i32, i32* %4
	ret i32 %23

24:
	%25 = load i32, i32* %6
	%26 = icmp eq i32 %25, 92
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	%29 = zext i1 %28 to i32
	%30 = icmp ne i32 %29, 0
	br i1 %30, label %31, label %43

dead29:
	br label %24

31:
	%32 = load i32, i32* %4
	%33 = add i32 %32, 1
	%34 = load i32*, i32** %2
	%35 = getelementptr i8, i32* %34, i32 %33
	%36 = load i8, i8* %35
	%37 = sext i8 %36 to i32
	%38 = and i32 %37, 255
	%39 = icmp ne i32 %38, 0
	%40 = zext i1 %39 to i32
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	br label %43

43:
	%44 = phi i32 [ %29, %24 ], [ %42, %31 ]
	%45 = icmp ne i32 %44, 0
	br i1 %45, label %46, label %49

46:
	%47 = load i32, i32* %4
	%48 = add i32 %47, 2
	store i32 %48, i32* %4
	br label %8

49:
	%50 = load i32, i32* %6
	%51 = icmp eq i32 %50, 40
	%52 = zext i1 %51 to i32
	%53 = icmp ne i32 %52, 0
	br i1 %53, label %54, label %60

dead30:
	br label %49

54:
	%55 = load i32, i32* %5
	%56 = add i32 %55, 1
	store i32 %56, i32* %5
	br label %57

57:
	%58 = load i32, i32* %4
	%59 = add i32 %58, 1
	store i32 %59, i32* %4
	br label %8

60:
	%61 = load i32, i32* %6
	%62 = icmp eq i32 %61, 41
	%63 = zext i1 %62 to i32
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %65, label %71

65:
	%66 = load i32, i32* %5
	%67 = icmp eq i32 %66, 0
	%68 = zext i1 %67 to i32
	%69 = icmp ne i32 %68, 0
	br i1 %69, label %78, label %80

70:
	br label %57

71:
	%72 = load i32, i32* %6
	%73 = icmp eq i32 %72, 124
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	%76 = zext i1 %75 to i32
	%77 = icmp ne i32 %76, 0
	br i1 %77, label %83, label %89

78:
	%79 = load i32, i32* %4
	ret i32 %79

80:
	%81 = load i32, i32* %5
	%82 = sub i32 %81, 1
	store i32 %82, i32* %5
	br label %70

dead31:
	br label %80

83:
	%84 = load i32, i32* %5
	%85 = icmp eq i32 %84, 0
	%86 = zext i1 %85 to i32
	%87 = icmp ne i32 %86, 0
	%88 = zext i1 %87 to i32
	br label %89

89:
	%90 = phi i32 [ %76, %71 ], [ %88, %83 ]
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %92, label %94

92:
	%93 = load i32, i32* %4
	ret i32 %93

94:
	br label %70

dead32:
	br label %94
}

define i32 @rt_glob(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32*
	%5 = alloca i32*
	%6 = alloca i32*
	%7 = alloca i32*
	%8 = alloca i32*
	%9 = alloca i32*
	%10 = alloca i32*
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = alloca i32
	%15 = alloca i32
	%16 = alloca i32
	%17 = alloca i32
	%18 = alloca i32
	%19 = alloca i32
	%20 = alloca i32
	%21 = alloca i32
	%22 = alloca i32
	%23 = alloca i32
	%24 = load i32*, i32** %2
	%25 = bitcast i32* %24 to i32*
	store i32* %25, i32** %4
	%26 = load i32*, i32** %3
	%27 = bitcast i32* %26 to i32*
	store i32* %27, i32** %5
	br label %28

28:
	%29 = icmp ne i32 1, 0
	br i1 %29, label %30, label %42

30:
	%31 = load i32*, i32** %4
	%32 = bitcast i32* %31 to i32*
	store i32* %32, i32** %6
	%33 = load i32*, i32** %6
	%34 = getelementptr i8, i32* %33, i32 0
	%35 = load i8, i8* %34
	%36 = sext i8 %35 to i32
	%37 = and i32 %36, 255
	store i32 %37, i32* %11
	%38 = load i32, i32* %11
	%39 = icmp eq i32 %38, 0
	%40 = zext i1 %39 to i32
	%41 = icmp ne i32 %40, 0
	br i1 %41, label %43, label %53

42:
	ret i32 0

43:
	%44 = load i32*, i32** %5
	%45 = getelementptr i8, i32* %44, i32 0
	%46 = load i8, i8* %45
	%47 = sext i8 %46 to i32
	%48 = and i32 %47, 255
	store i32 %48, i32* %14
	%49 = load i32, i32* %14
	%50 = icmp eq i32 %49, 0
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %60, label %61

53:
	%54 = load i32, i32* %11
	%55 = icmp eq i32 %54, 64
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	%58 = zext i1 %57 to i32
	%59 = icmp ne i32 %58, 0
	br i1 %59, label %68, label %62

60:
	ret i32 1

61:
	ret i32 0

dead33:
	br label %61

dead34:
	br label %53

62:
	%63 = load i32, i32* %11
	%64 = icmp eq i32 %63, 63
	%65 = zext i1 %64 to i32
	%66 = icmp ne i32 %65, 0
	%67 = zext i1 %66 to i32
	br label %68

68:
	%69 = phi i32 [ %58, %53 ], [ %67, %62 ]
	%70 = icmp ne i32 %69, 0
	br i1 %70, label %77, label %71

71:
	%72 = load i32, i32* %11
	%73 = icmp eq i32 %72, 42
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	%76 = zext i1 %75 to i32
	br label %77

77:
	%78 = phi i32 [ %69, %68 ], [ %76, %71 ]
	%79 = icmp ne i32 %78, 0
	br i1 %79, label %86, label %80

80:
	%81 = load i32, i32* %11
	%82 = icmp eq i32 %81, 43
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	%85 = zext i1 %84 to i32
	br label %86

86:
	%87 = phi i32 [ %78, %77 ], [ %85, %80 ]
	%88 = icmp ne i32 %87, 0
	br i1 %88, label %95, label %89

89:
	%90 = load i32, i32* %11
	%91 = icmp eq i32 %90, 33
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	%94 = zext i1 %93 to i32
	br label %95

95:
	%96 = phi i32 [ %87, %86 ], [ %94, %89 ]
	store i32 %96, i32* %12
	%97 = load i32*, i32** %6
	%98 = getelementptr i8, i32* %97, i32 1
	%99 = load i8, i8* %98
	%100 = sext i8 %99 to i32
	%101 = and i32 %100, 255
	store i32 %101, i32* %13
	%102 = load i32, i32* %12
	%103 = icmp ne i32 %102, 0
	%104 = zext i1 %103 to i32
	%105 = icmp ne i32 %104, 0
	br i1 %105, label %106, label %112

106:
	%107 = load i32, i32* %13
	%108 = icmp eq i32 %107, 40
	%109 = zext i1 %108 to i32
	%110 = icmp ne i32 %109, 0
	%111 = zext i1 %110 to i32
	br label %112

112:
	%113 = phi i32 [ %104, %95 ], [ %111, %106 ]
	%114 = icmp ne i32 %113, 0
	br i1 %114, label %115, label %121

115:
	%116 = load i32*, i32** %6
	%117 = bitcast i32* %116 to i32*
	%118 = load i32*, i32** %5
	%119 = bitcast i32* %118 to i32*
	%120 = call i32 @rt_eg(i32* %117, i32* %119, i32 0)
	ret i32 %120

121:
	%122 = load i32, i32* %11
	%123 = icmp eq i32 %122, 42
	%124 = zext i1 %123 to i32
	%125 = icmp ne i32 %124, 0
	br i1 %125, label %126, label %127

dead35:
	br label %121

126:
	br label %139

127:
	%128 = load i32*, i32** %5
	%129 = bitcast i32* %128 to i32*
	store i32* %129, i32** %10
	%130 = load i32*, i32** %10
	%131 = getelementptr i8, i32* %130, i32 0
	%132 = load i8, i8* %131
	%133 = sext i8 %132 to i32
	%134 = and i32 %133, 255
	store i32 %134, i32* %18
	%135 = load i32, i32* %18
	%136 = icmp eq i32 %135, 0
	%137 = zext i1 %136 to i32
	%138 = icmp ne i32 %137, 0
	br i1 %138, label %204, label %205

139:
	%140 = icmp ne i32 1, 0
	br i1 %140, label %141, label %152

141:
	%142 = load i32*, i32** %4
	%143 = bitcast i32* %142 to i32*
	store i32* %143, i32** %7
	%144 = load i32*, i32** %7
	%145 = getelementptr i8, i32* %144, i32 0
	%146 = load i8, i8* %145
	%147 = sext i8 %146 to i32
	%148 = and i32 %147, 255
	%149 = icmp ne i32 %148, 42
	%150 = zext i1 %149 to i32
	%151 = icmp ne i32 %150, 0
	br i1 %151, label %164, label %165

152:
	%153 = load i32*, i32** %4
	%154 = bitcast i32* %153 to i32*
	store i32* %154, i32** %8
	%155 = load i32*, i32** %8
	%156 = getelementptr i8, i32* %155, i32 0
	%157 = load i8, i8* %156
	%158 = sext i8 %157 to i32
	%159 = and i32 %158, 255
	store i32 %159, i32* %15
	%160 = load i32, i32* %15
	%161 = icmp eq i32 %160, 0
	%162 = zext i1 %161 to i32
	%163 = icmp ne i32 %162, 0
	br i1 %163, label %170, label %171

164:
	br label %152

165:
	%166 = load i32*, i32** %7
	%167 = sext i32 1 to i64
	%168 = getelementptr i8, i32* %166, i64 %167
	%169 = bitcast i8* %168 to i32*
	store i32* %169, i32** %4
	br label %139

dead36:
	br label %165

170:
	ret i32 1

171:
	br label %172

dead37:
	br label %171

172:
	%173 = icmp ne i32 1, 0
	br i1 %173, label %174, label %186

174:
	%175 = load i32*, i32** %5
	%176 = bitcast i32* %175 to i32*
	store i32* %176, i32** %9
	%177 = load i32*, i32** %8
	%178 = bitcast i32* %177 to i32*
	%179 = load i32*, i32** %9
	%180 = bitcast i32* %179 to i32*
	%181 = call i32 @rt_glob(i32* %178, i32* %180)
	store i32 %181, i32* %16
	%182 = load i32, i32* %16
	%183 = icmp ne i32 %182, 0
	%184 = zext i1 %183 to i32
	%185 = icmp ne i32 %184, 0
	br i1 %185, label %187, label %188

186:
	br label %127

187:
	ret i32 1

188:
	%189 = load i32*, i32** %9
	%190 = getelementptr i8, i32* %189, i32 0
	%191 = load i8, i8* %190
	%192 = sext i8 %191 to i32
	%193 = and i32 %192, 255
	store i32 %193, i32* %17
	%194 = load i32, i32* %17
	%195 = icmp eq i32 %194, 0
	%196 = zext i1 %195 to i32
	%197 = icmp ne i32 %196, 0
	br i1 %197, label %198, label %199

dead38:
	br label %188

198:
	ret i32 0

199:
	%200 = load i32*, i32** %9
	%201 = sext i32 1 to i64
	%202 = getelementptr i8, i32* %200, i64 %201
	%203 = bitcast i8* %202 to i32*
	store i32* %203, i32** %5
	br label %172

dead39:
	br label %199

204:
	ret i32 0

205:
	%206 = load i32, i32* %11
	%207 = icmp eq i32 %206, 63
	%208 = zext i1 %207 to i32
	%209 = icmp ne i32 %208, 0
	br i1 %209, label %210, label %219

dead40:
	br label %205

210:
	%211 = load i32*, i32** %6
	%212 = sext i32 1 to i64
	%213 = getelementptr i8, i32* %211, i64 %212
	%214 = bitcast i8* %213 to i32*
	store i32* %214, i32** %4
	%215 = load i32*, i32** %10
	%216 = sext i32 1 to i64
	%217 = getelementptr i8, i32* %215, i64 %216
	%218 = bitcast i8* %217 to i32*
	store i32* %218, i32** %5
	br label %28

219:
	%220 = load i32, i32* %11
	%221 = icmp eq i32 %220, 91
	%222 = zext i1 %221 to i32
	%223 = icmp ne i32 %222, 0
	br i1 %223, label %224, label %239

dead41:
	br label %219

224:
	%225 = load i32*, i32** %6
	%226 = bitcast i32* %225 to i32*
	%227 = load i32, i32* %18
	%228 = call i32 @rt_class(i32* %226, i32 %227)
	store i32 %228, i32* %19
	%229 = load i32, i32* %19
	%230 = icmp sge i32 %229, 0
	%231 = zext i1 %230 to i32
	%232 = icmp ne i32 %231, 0
	br i1 %232, label %251, label %260

233:
	%234 = load i32, i32* %11
	%235 = load i32, i32* %18
	%236 = icmp ne i32 %234, %235
	%237 = zext i1 %236 to i32
	%238 = icmp ne i32 %237, 0
	br i1 %238, label %299, label %300

239:
	%240 = load i32*, i32** %6
	%241 = getelementptr i8, i32* %240, i32 1
	%242 = load i8, i8* %241
	%243 = sext i8 %242 to i32
	%244 = and i32 %243, 255
	store i32 %244, i32* %22
	%245 = load i32, i32* %11
	%246 = icmp eq i32 %245, 92
	%247 = zext i1 %246 to i32
	%248 = icmp ne i32 %247, 0
	%249 = zext i1 %248 to i32
	%250 = icmp ne i32 %249, 0
	br i1 %250, label %272, label %278

251:
	%252 = load i32, i32* %19
	%253 = and i32 %252, 1
	store i32 %253, i32* %20
	%254 = load i32, i32* %19
	%255 = sdiv i32 %254, 2
	store i32 %255, i32* %21
	%256 = load i32, i32* %20
	%257 = icmp eq i32 %256, 0
	%258 = zext i1 %257 to i32
	%259 = icmp ne i32 %258, 0
	br i1 %259, label %261, label %262

260:
	br label %233

261:
	ret i32 0

262:
	%263 = load i32*, i32** %6
	%264 = load i32, i32* %21
	%265 = sext i32 %264 to i64
	%266 = getelementptr i8, i32* %263, i64 %265
	%267 = bitcast i8* %266 to i32*
	store i32* %267, i32** %4
	%268 = load i32*, i32** %10
	%269 = sext i32 1 to i64
	%270 = getelementptr i8, i32* %268, i64 %269
	%271 = bitcast i8* %270 to i32*
	store i32* %271, i32** %5
	br label %28

dead42:
	br label %262

dead43:
	br label %260

272:
	%273 = load i32, i32* %22
	%274 = icmp ne i32 %273, 0
	%275 = zext i1 %274 to i32
	%276 = icmp ne i32 %275, 0
	%277 = zext i1 %276 to i32
	br label %278

278:
	%279 = phi i32 [ %249, %239 ], [ %277, %272 ]
	store i32 %279, i32* %23
	%280 = load i32, i32* %23
	%281 = icmp ne i32 %280, 0
	br i1 %281, label %282, label %288

282:
	%283 = load i32, i32* %22
	%284 = load i32, i32* %18
	%285 = icmp ne i32 %283, %284
	%286 = zext i1 %285 to i32
	%287 = icmp ne i32 %286, 0
	br i1 %287, label %289, label %290

288:
	br label %233

289:
	ret i32 0

290:
	%291 = load i32*, i32** %6
	%292 = sext i32 2 to i64
	%293 = getelementptr i8, i32* %291, i64 %292
	%294 = bitcast i8* %293 to i32*
	store i32* %294, i32** %4
	%295 = load i32*, i32** %10
	%296 = sext i32 1 to i64
	%297 = getelementptr i8, i32* %295, i64 %296
	%298 = bitcast i8* %297 to i32*
	store i32* %298, i32** %5
	br label %28

dead44:
	br label %290

dead45:
	br label %288

299:
	ret i32 0

300:
	%301 = load i32*, i32** %6
	%302 = sext i32 1 to i64
	%303 = getelementptr i8, i32* %301, i64 %302
	%304 = bitcast i8* %303 to i32*
	store i32* %304, i32** %4
	%305 = load i32*, i32** %10
	%306 = sext i32 1 to i64
	%307 = getelementptr i8, i32* %305, i64 %306
	%308 = bitcast i8* %307 to i32*
	store i32* %308, i32** %5
	br label %28

dead46:
	br label %300
}

define i32 @rt_eg(i32* %0, i32* %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32*
	store i32* %1, i32** %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = alloca i32*
	%8 = alloca i32
	%9 = alloca i32*
	%10 = alloca i32
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = alloca i32
	%15 = alloca i32
	%16 = alloca i32
	%17 = alloca i32
	%18 = alloca i32
	%19 = alloca i32*
	%20 = alloca i32
	%21 = alloca i32*
	%22 = alloca i32*
	%23 = alloca i32*
	%24 = alloca i32
	%25 = alloca i32
	%26 = alloca i32
	%27 = alloca i32*
	%28 = alloca i32*
	%29 = alloca i32
	%30 = alloca i32*
	%31 = alloca i32
	%32 = load i32, i32* %5
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %37

36:
	br label %43

37:
	%38 = load i32*, i32** %3
	%39 = getelementptr i8, i32* %38, i32 0
	%40 = load i8, i8* %39
	%41 = sext i8 %40 to i32
	%42 = and i32 %41, 255
	br label %43

43:
	%44 = phi i32 [ 42, %36 ], [ %42, %37 ]
	store i32 %44, i32* %6
	%45 = load i32*, i32** %3
	%46 = sext i32 1 to i64
	%47 = getelementptr i8, i32* %45, i64 %46
	%48 = bitcast i8* %47 to i32*
	store i32* %48, i32** %7
	%49 = load i32*, i32** %7
	%50 = bitcast i32* %49 to i32*
	%51 = call i32 @rt_egclose(i32* %50)
	store i32 %51, i32* %8
	%52 = load i32, i32* %8
	%53 = icmp slt i32 %52, 0
	%54 = zext i1 %53 to i32
	%55 = icmp ne i32 %54, 0
	br i1 %55, label %56, label %57

56:
	ret i32 0

57:
	%58 = load i32*, i32** %3
	%59 = load i32, i32* %8
	%60 = add i32 %59, 2
	%61 = sext i32 %60 to i64
	%62 = getelementptr i8, i32* %58, i64 %61
	%63 = bitcast i8* %62 to i32*
	store i32* %63, i32** %9
	%64 = load i32*, i32** %4
	%65 = bitcast i32* %64 to i32*
	%66 = call i32 @rt_strlen(i32* %65)
	store i32 %66, i32* %10
	%67 = load i32, i32* %6
	%68 = icmp eq i32 %67, 33
	%69 = zext i1 %68 to i32
	store i32 %69, i32* %11
	%70 = load i32, i32* %6
	%71 = icmp eq i32 %70, 64
	%72 = zext i1 %71 to i32
	store i32 %72, i32* %12
	%73 = load i32, i32* %6
	%74 = icmp eq i32 %73, 63
	%75 = zext i1 %74 to i32
	store i32 %75, i32* %13
	%76 = load i32, i32* %13
	%77 = icmp ne i32 %76, 0
	%78 = zext i1 %77 to i32
	%79 = icmp ne i32 %78, 0
	br i1 %79, label %86, label %80

dead47:
	br label %57

80:
	%81 = load i32, i32* %6
	%82 = icmp eq i32 %81, 42
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	%85 = zext i1 %84 to i32
	br label %86

86:
	%87 = phi i32 [ %78, %57 ], [ %85, %80 ]
	store i32 %87, i32* %14
	%88 = load i32, i32* %12
	%89 = icmp ne i32 %88, 0
	%90 = zext i1 %89 to i32
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %96, label %92

92:
	%93 = load i32, i32* %13
	%94 = icmp ne i32 %93, 0
	%95 = zext i1 %94 to i32
	br label %96

96:
	%97 = phi i32 [ %90, %86 ], [ %95, %92 ]
	store i32 %97, i32* %15
	%98 = load i32, i32* %11
	%99 = icmp ne i32 %98, 0
	br i1 %99, label %100, label %101

100:
	store i32 0, i32* %20
	br label %104

101:
	%102 = load i32, i32* %14
	%103 = icmp ne i32 %102, 0
	br i1 %103, label %195, label %204

104:
	%105 = load i32, i32* %20
	%106 = load i32, i32* %10
	%107 = add i32 %106, 1
	%108 = icmp slt i32 %105, %107
	%109 = zext i1 %108 to i32
	%110 = icmp ne i32 %109, 0
	br i1 %110, label %111, label %112

111:
	store i32 0, i32* %29
	store i32 1, i32* %25
	br label %113

112:
	ret i32 0

113:
	%114 = load i32, i32* %25
	%115 = load i32, i32* %8
	%116 = icmp slt i32 %114, %115
	%117 = zext i1 %116 to i32
	%118 = icmp ne i32 %117, 0
	br i1 %118, label %119, label %147

119:
	%120 = load i32*, i32** %7
	%121 = bitcast i32* %120 to i32*
	%122 = load i32, i32* %25
	%123 = call i32 @rt_egalt(i32* %121, i32 %122)
	store i32 %123, i32* %26
	%124 = load i32*, i32** %7
	%125 = load i32, i32* %25
	%126 = sext i32 %125 to i64
	%127 = getelementptr i8, i32* %124, i64 %126
	%128 = bitcast i8* %127 to i32*
	%129 = load i32, i32* %26
	%130 = load i32, i32* %25
	%131 = sub i32 %129, %130
	%132 = call i32* @rt_substr(i32* %128, i32 0, i32 %131)
	%133 = bitcast i32* %132 to i32*
	store i32* %133, i32** %27
	%134 = load i32*, i32** %4
	%135 = bitcast i32* %134 to i32*
	%136 = load i32, i32* %20
	%137 = call i32* @rt_substr(i32* %135, i32 0, i32 %136)
	%138 = bitcast i32* %137 to i32*
	store i32* %138, i32** %28
	%139 = load i32*, i32** %27
	%140 = bitcast i32* %139 to i32*
	%141 = load i32*, i32** %28
	%142 = bitcast i32* %141 to i32*
	%143 = call i32 @rt_glob(i32* %140, i32* %142)
	%144 = icmp ne i32 %143, 0
	%145 = zext i1 %144 to i32
	%146 = icmp ne i32 %145, 0
	br i1 %146, label %164, label %165

147:
	%148 = load i32*, i32** %4
	%149 = load i32, i32* %20
	%150 = sext i32 %149 to i64
	%151 = getelementptr i8, i32* %148, i64 %150
	%152 = bitcast i8* %151 to i32*
	store i32* %152, i32** %30
	%153 = load i32*, i32** %9
	%154 = bitcast i32* %153 to i32*
	%155 = load i32*, i32** %30
	%156 = bitcast i32* %155 to i32*
	%157 = call i32 @rt_glob(i32* %154, i32* %156)
	store i32 %157, i32* %31
	%158 = load i32, i32* %29
	%159 = icmp eq i32 %158, 0
	%160 = zext i1 %159 to i32
	%161 = icmp ne i32 %160, 0
	%162 = zext i1 %161 to i32
	%163 = icmp ne i32 %162, 0
	br i1 %163, label %182, label %188

164:
	br label %167

165:
	%166 = load i32, i32* %29
	br label %167

167:
	%168 = phi i32 [ 1, %164 ], [ %166, %165 ]
	store i32 %168, i32* %29
	%169 = load i32, i32* %26
	%170 = load i32*, i32** %7
	%171 = getelementptr i8, i32* %170, i32 %169
	%172 = load i8, i8* %171
	%173 = sext i8 %172 to i32
	%174 = and i32 %173, 255
	%175 = icmp eq i32 %174, 41
	%176 = zext i1 %175 to i32
	%177 = icmp ne i32 %176, 0
	br i1 %177, label %178, label %179

178:
	br label %147

179:
	%180 = load i32, i32* %26
	%181 = add i32 %180, 1
	store i32 %181, i32* %25
	br label %113

dead48:
	br label %179

182:
	%183 = load i32, i32* %31
	%184 = icmp ne i32 %183, 0
	%185 = zext i1 %184 to i32
	%186 = icmp ne i32 %185, 0
	%187 = zext i1 %186 to i32
	br label %188

188:
	%189 = phi i32 [ %162, %147 ], [ %187, %182 ]
	%190 = icmp ne i32 %189, 0
	br i1 %190, label %191, label %192

191:
	ret i32 1

192:
	%193 = load i32, i32* %20
	%194 = add i32 %193, 1
	store i32 %194, i32* %20
	br label %104

dead49:
	br label %192

dead50:
	br label %101

195:
	%196 = load i32*, i32** %9
	%197 = bitcast i32* %196 to i32*
	%198 = load i32*, i32** %4
	%199 = bitcast i32* %198 to i32*
	%200 = call i32 @rt_glob(i32* %197, i32* %199)
	%201 = icmp ne i32 %200, 0
	%202 = zext i1 %201 to i32
	%203 = icmp ne i32 %202, 0
	br i1 %203, label %207, label %208

204:
	%205 = load i32, i32* %15
	%206 = icmp ne i32 %205, 0
	br i1 %206, label %209, label %210

207:
	ret i32 1

208:
	br label %204

dead51:
	br label %208

209:
	br label %211

210:
	br label %211

211:
	%212 = phi i32 [ 0, %209 ], [ 1, %210 ]
	store i32 %212, i32* %16
	store i32 1, i32* %17
	br label %213

213:
	%214 = load i32, i32* %17
	%215 = load i32, i32* %8
	%216 = icmp slt i32 %214, %215
	%217 = zext i1 %216 to i32
	%218 = icmp ne i32 %217, 0
	br i1 %218, label %219, label %235

219:
	%220 = load i32*, i32** %7
	%221 = bitcast i32* %220 to i32*
	%222 = load i32, i32* %17
	%223 = call i32 @rt_egalt(i32* %221, i32 %222)
	store i32 %223, i32* %18
	%224 = load i32*, i32** %7
	%225 = load i32, i32* %17
	%226 = sext i32 %225 to i64
	%227 = getelementptr i8, i32* %224, i64 %226
	%228 = bitcast i8* %227 to i32*
	%229 = load i32, i32* %18
	%230 = load i32, i32* %17
	%231 = sub i32 %229, %230
	%232 = call i32* @rt_substr(i32* %228, i32 0, i32 %231)
	%233 = bitcast i32* %232 to i32*
	store i32* %233, i32** %19
	%234 = load i32, i32* %16
	store i32 %234, i32* %20
	br label %236

235:
	ret i32 0

236:
	%237 = load i32, i32* %20
	%238 = load i32, i32* %10
	%239 = add i32 %238, 1
	%240 = icmp slt i32 %237, %239
	%241 = zext i1 %240 to i32
	%242 = icmp ne i32 %241, 0
	br i1 %242, label %243, label %257

243:
	%244 = load i32*, i32** %4
	%245 = bitcast i32* %244 to i32*
	%246 = load i32, i32* %20
	%247 = call i32* @rt_substr(i32* %245, i32 0, i32 %246)
	%248 = bitcast i32* %247 to i32*
	store i32* %248, i32** %21
	%249 = load i32*, i32** %19
	%250 = bitcast i32* %249 to i32*
	%251 = load i32*, i32** %21
	%252 = bitcast i32* %251 to i32*
	%253 = call i32 @rt_glob(i32* %250, i32* %252)
	%254 = icmp ne i32 %253, 0
	%255 = zext i1 %254 to i32
	%256 = icmp ne i32 %255, 0
	br i1 %256, label %267, label %270

257:
	%258 = load i32, i32* %18
	%259 = load i32*, i32** %7
	%260 = getelementptr i8, i32* %259, i32 %258
	%261 = load i8, i8* %260
	%262 = sext i8 %261 to i32
	%263 = and i32 %262, 255
	%264 = icmp eq i32 %263, 41
	%265 = zext i1 %264 to i32
	%266 = icmp ne i32 %265, 0
	br i1 %266, label %318, label %319

267:
	%268 = load i32, i32* %15
	%269 = icmp ne i32 %268, 0
	br i1 %269, label %273, label %288

270:
	%271 = load i32, i32* %20
	%272 = add i32 %271, 1
	store i32 %272, i32* %20
	br label %236

273:
	%274 = load i32*, i32** %4
	%275 = load i32, i32* %20
	%276 = sext i32 %275 to i64
	%277 = getelementptr i8, i32* %274, i64 %276
	%278 = bitcast i8* %277 to i32*
	store i32* %278, i32** %22
	%279 = load i32*, i32** %9
	%280 = bitcast i32* %279 to i32*
	%281 = load i32*, i32** %22
	%282 = bitcast i32* %281 to i32*
	%283 = call i32 @rt_glob(i32* %280, i32* %282)
	%284 = icmp ne i32 %283, 0
	%285 = zext i1 %284 to i32
	%286 = icmp ne i32 %285, 0
	br i1 %286, label %305, label %306

287:
	br label %270

288:
	%289 = load i32*, i32** %4
	%290 = load i32, i32* %20
	%291 = sext i32 %290 to i64
	%292 = getelementptr i8, i32* %289, i64 %291
	%293 = bitcast i8* %292 to i32*
	store i32* %293, i32** %23
	%294 = load i32*, i32** %3
	%295 = bitcast i32* %294 to i32*
	%296 = load i32*, i32** %23
	%297 = bitcast i32* %296 to i32*
	%298 = call i32 @rt_eg(i32* %295, i32* %297, i32 1)
	store i32 %298, i32* %24
	%299 = load i32, i32* %20
	%300 = icmp sgt i32 %299, 0
	%301 = zext i1 %300 to i32
	%302 = icmp ne i32 %301, 0
	%303 = zext i1 %302 to i32
	%304 = icmp ne i32 %303, 0
	br i1 %304, label %307, label %313

305:
	ret i32 1

306:
	br label %287

dead52:
	br label %306

307:
	%308 = load i32, i32* %24
	%309 = icmp ne i32 %308, 0
	%310 = zext i1 %309 to i32
	%311 = icmp ne i32 %310, 0
	%312 = zext i1 %311 to i32
	br label %313

313:
	%314 = phi i32 [ %303, %288 ], [ %312, %307 ]
	%315 = icmp ne i32 %314, 0
	br i1 %315, label %316, label %317

316:
	ret i32 1

317:
	br label %287

dead53:
	br label %317

318:
	br label %235

319:
	%320 = load i32, i32* %18
	%321 = add i32 %320, 1
	store i32 %321, i32* %17
	br label %213

dead54:
	br label %319

dead55:
	ret i32 0
}

define i32 @rt_matchlen(i32* %0, i32* %1, i32 %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32*
	store i32* %1, i32** %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = alloca i32*
	%8 = load i32*, i32** %4
	%9 = bitcast i32* %8 to i32*
	%10 = call i32 @rt_strlen(i32* %9)
	store i32 %10, i32* %6
	br label %11

11:
	%12 = icmp ne i32 1, 0
	br i1 %12, label %13, label %19

13:
	%14 = load i32, i32* %6
	%15 = load i32, i32* %5
	%16 = icmp slt i32 %14, %15
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %20, label %21

19:
	ret i32 0

20:
	ret i32 -1

21:
	%22 = load i32*, i32** %4
	%23 = bitcast i32* %22 to i32*
	%24 = load i32, i32* %5
	%25 = load i32, i32* %6
	%26 = load i32, i32* %5
	%27 = sub i32 %25, %26
	%28 = call i32* @rt_substr(i32* %23, i32 %24, i32 %27)
	%29 = bitcast i32* %28 to i32*
	store i32* %29, i32** %7
	%30 = load i32*, i32** %3
	%31 = bitcast i32* %30 to i32*
	%32 = load i32*, i32** %7
	%33 = bitcast i32* %32 to i32*
	%34 = call i32 @rt_glob(i32* %31, i32* %33)
	%35 = icmp ne i32 %34, 0
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %38, label %40

dead56:
	br label %21

38:
	%39 = load i32, i32* %6
	ret i32 %39

40:
	%41 = load i32, i32* %6
	%42 = sub i32 %41, 1
	store i32 %42, i32* %6
	br label %11

dead57:
	br label %40
}

define i32 @rt_matchend(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32*
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_strlen(i32* %8)
	store i32 %9, i32* %4
	store i32 0, i32* %5
	br label %10

10:
	%11 = icmp ne i32 1, 0
	br i1 %11, label %12, label %18

12:
	%13 = load i32, i32* %5
	%14 = load i32, i32* %4
	%15 = icmp sgt i32 %13, %14
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %19, label %20

18:
	ret i32 0

19:
	ret i32 -1

20:
	%21 = load i32*, i32** %3
	%22 = bitcast i32* %21 to i32*
	%23 = load i32, i32* %5
	%24 = call i32* @rt_substr(i32* %22, i32 %23, i32 -1)
	%25 = bitcast i32* %24 to i32*
	store i32* %25, i32** %6
	%26 = load i32*, i32** %2
	%27 = bitcast i32* %26 to i32*
	%28 = load i32*, i32** %6
	%29 = bitcast i32* %28 to i32*
	%30 = call i32 @rt_glob(i32* %27, i32* %29)
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	br i1 %33, label %34, label %36

dead58:
	br label %20

34:
	%35 = load i32, i32* %5
	ret i32 %35

36:
	%37 = load i32, i32* %5
	%38 = add i32 %37, 1
	store i32 %38, i32* %5
	br label %10

dead59:
	br label %36
}

define i32* @rt_strip(i32* %0, i32* %1, i32 %2) {
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
	%11 = load i32, i32* %5
	%12 = icmp eq i32 %11, 0
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	%15 = zext i1 %14 to i32
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %23, label %17

17:
	%18 = load i32, i32* %5
	%19 = icmp eq i32 %18, 1
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	br label %23

23:
	%24 = phi i32 [ %15, %entry ], [ %22, %17 ]
	store i32 %24, i32* %10
	%25 = alloca i32
	%26 = load i32, i32* %5
	%27 = icmp eq i32 %26, 0
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	%30 = zext i1 %29 to i32
	%31 = icmp ne i32 %30, 0
	br i1 %31, label %38, label %32

32:
	%33 = load i32, i32* %5
	%34 = icmp eq i32 %33, 3
	%35 = zext i1 %34 to i32
	%36 = icmp ne i32 %35, 0
	%37 = zext i1 %36 to i32
	br label %38

38:
	%39 = phi i32 [ %30, %23 ], [ %37, %32 ]
	store i32 %39, i32* %25
	%40 = alloca i32
	store i32 0, i32* %40
	br label %41

41:
	%42 = load i32, i32* %40
	%43 = load i32, i32* %6
	%44 = icmp sle i32 %42, %43
	%45 = zext i1 %44 to i32
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %51

47:
	%48 = alloca i32
	%49 = load i32, i32* %25
	%50 = icmp ne i32 %49, 0
	br i1 %50, label %54, label %56

51:
	%52 = load i32*, i32** %3
	%53 = bitcast i32* %52 to i32*
	ret i32* %53

54:
	%55 = load i32, i32* %40
	br label %60

56:
	%57 = load i32, i32* %6
	%58 = load i32, i32* %40
	%59 = sub i32 %57, %58
	br label %60

60:
	%61 = phi i32 [ %55, %54 ], [ %59, %56 ]
	store i32 %61, i32* %48
	%62 = alloca i32*
	%63 = load i32*, i32** %3
	%64 = bitcast i32* %63 to i32*
	%65 = load i32, i32* %48
	%66 = call i32* @rt_substr(i32* %64, i32 0, i32 %65)
	%67 = bitcast i32* %66 to i32*
	store i32* %67, i32** %62
	%68 = alloca i32*
	%69 = load i32*, i32** %3
	%70 = bitcast i32* %69 to i32*
	%71 = load i32, i32* %48
	%72 = call i32* @rt_substr(i32* %70, i32 %71, i32 -1)
	%73 = bitcast i32* %72 to i32*
	store i32* %73, i32** %68
	%74 = alloca i32*
	%75 = load i32, i32* %10
	%76 = icmp ne i32 %75, 0
	br i1 %76, label %77, label %80

77:
	%78 = load i32*, i32** %62
	%79 = bitcast i32* %78 to i32*
	br label %83

80:
	%81 = load i32*, i32** %68
	%82 = bitcast i32* %81 to i32*
	br label %83

83:
	%84 = phi i32* [ %79, %77 ], [ %82, %80 ]
	%85 = bitcast i32* %84 to i32*
	store i32* %85, i32** %74
	%86 = load i32*, i32** %4
	%87 = bitcast i32* %86 to i32*
	%88 = load i32*, i32** %74
	%89 = bitcast i32* %88 to i32*
	%90 = call i32 @rt_glob(i32* %87, i32* %89)
	%91 = icmp ne i32 %90, 0
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	br i1 %93, label %94, label %109

94:
	%95 = alloca i32*
	%96 = load i32*, i32** %3
	%97 = bitcast i32* %96 to i32*
	%98 = load i32, i32* %48
	%99 = call i32* @rt_substr(i32* %97, i32 %98, i32 -1)
	%100 = bitcast i32* %99 to i32*
	store i32* %100, i32** %95
	%101 = alloca i32*
	%102 = load i32*, i32** %3
	%103 = bitcast i32* %102 to i32*
	%104 = load i32, i32* %48
	%105 = call i32* @rt_substr(i32* %103, i32 0, i32 %104)
	%106 = bitcast i32* %105 to i32*
	store i32* %106, i32** %101
	%107 = load i32, i32* %10
	%108 = icmp ne i32 %107, 0
	br i1 %108, label %112, label %115

109:
	%110 = load i32, i32* %40
	%111 = add i32 %110, 1
	store i32 %111, i32* %40
	br label %41

112:
	%113 = load i32*, i32** %95
	%114 = bitcast i32* %113 to i32*
	br label %118

115:
	%116 = load i32*, i32** %101
	%117 = bitcast i32* %116 to i32*
	br label %118

118:
	%119 = phi i32* [ %114, %112 ], [ %117, %115 ]
	%120 = bitcast i32* %119 to i32*
	ret i32* %120

dead60:
	br label %109

dead61:
	ret i32* null
}

define i32* @rt_replace(i32* %0, i32* %1, i32* %2, i32 %3) {
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
	%9 = load i32*, i32** %4
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_strlen(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32*
	%13 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%14 = bitcast i8* %13 to i32*
	store i32* %14, i32** %12
	%15 = alloca i32
	store i32 0, i32* %15
	%16 = alloca i32
	store i32 0, i32* %16
	%17 = load i32, i32* %7
	%18 = icmp sge i32 %17, 2
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %26

21:
	%22 = load i32, i32* %7
	%23 = icmp eq i32 %22, 2
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %27, label %38

26:
	br label %75

27:
	%28 = alloca i32
	%29 = load i32*, i32** %5
	%30 = bitcast i32* %29 to i32*
	%31 = load i32*, i32** %4
	%32 = bitcast i32* %31 to i32*
	%33 = call i32 @rt_matchlen(i32* %30, i32* %32, i32 0)
	store i32 %33, i32* %28
	%34 = load i32, i32* %28
	%35 = icmp slt i32 %34, 0
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %49, label %52

38:
	%39 = alloca i32
	%40 = load i32*, i32** %5
	%41 = bitcast i32* %40 to i32*
	%42 = load i32*, i32** %4
	%43 = bitcast i32* %42 to i32*
	%44 = call i32 @rt_matchend(i32* %41, i32* %43)
	store i32 %44, i32* %39
	%45 = load i32, i32* %39
	%46 = icmp slt i32 %45, 0
	%47 = zext i1 %46 to i32
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %62, label %65

49:
	%50 = load i32*, i32** %4
	%51 = bitcast i32* %50 to i32*
	ret i32* %51

52:
	%53 = load i32*, i32** %6
	%54 = bitcast i32* %53 to i32*
	%55 = load i32*, i32** %4
	%56 = bitcast i32* %55 to i32*
	%57 = load i32, i32* %28
	%58 = call i32* @rt_substr(i32* %56, i32 %57, i32 -1)
	%59 = bitcast i32* %58 to i32*
	%60 = call i32* @rt_strcat(i32* %54, i32* %59)
	%61 = bitcast i32* %60 to i32*
	ret i32* %61

dead62:
	br label %52

dead63:
	br label %38

62:
	%63 = load i32*, i32** %4
	%64 = bitcast i32* %63 to i32*
	ret i32* %64

65:
	%66 = load i32*, i32** %4
	%67 = bitcast i32* %66 to i32*
	%68 = load i32, i32* %39
	%69 = call i32* @rt_substr(i32* %67, i32 0, i32 %68)
	%70 = bitcast i32* %69 to i32*
	%71 = load i32*, i32** %6
	%72 = bitcast i32* %71 to i32*
	%73 = call i32* @rt_strcat(i32* %70, i32* %72)
	%74 = bitcast i32* %73 to i32*
	ret i32* %74

dead64:
	br label %65

dead65:
	br label %26

75:
	%76 = icmp ne i32 1, 0
	br i1 %76, label %77, label %85

77:
	%78 = load i32, i32* %15
	%79 = load i32, i32* %8
	%80 = icmp sgt i32 %78, %79
	%81 = zext i1 %80 to i32
	%82 = icmp ne i32 %81, 0
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %94, label %88

85:
	%86 = load i32*, i32** %12
	%87 = bitcast i32* %86 to i32*
	ret i32* %87

88:
	%89 = load i32, i32* %16
	%90 = icmp ne i32 %89, 0
	%91 = zext i1 %90 to i32
	%92 = icmp ne i32 %91, 0
	%93 = zext i1 %92 to i32
	br label %94

94:
	%95 = phi i32 [ %83, %77 ], [ %93, %88 ]
	%96 = icmp ne i32 %95, 0
	br i1 %96, label %97, label %100

97:
	%98 = load i32*, i32** %12
	%99 = bitcast i32* %98 to i32*
	ret i32* %99

100:
	%101 = alloca i32
	%102 = load i32*, i32** %5
	%103 = bitcast i32* %102 to i32*
	%104 = load i32*, i32** %4
	%105 = bitcast i32* %104 to i32*
	%106 = load i32, i32* %15
	%107 = call i32 @rt_matchlen(i32* %103, i32* %105, i32 %106)
	store i32 %107, i32* %101
	%108 = load i32, i32* %101
	%109 = load i32, i32* %15
	%110 = icmp sgt i32 %108, %109
	%111 = zext i1 %110 to i32
	%112 = icmp ne i32 %111, 0
	br i1 %112, label %113, label %126

dead66:
	br label %100

113:
	%114 = load i32*, i32** %12
	%115 = bitcast i32* %114 to i32*
	%116 = load i32*, i32** %6
	%117 = bitcast i32* %116 to i32*
	%118 = call i32* @rt_strcat(i32* %115, i32* %117)
	%119 = bitcast i32* %118 to i32*
	store i32* %119, i32** %12
	%120 = load i32, i32* %101
	store i32 %120, i32* %15
	%121 = load i32, i32* %7
	%122 = icmp eq i32 %121, 0
	%123 = zext i1 %122 to i32
	%124 = icmp ne i32 %123, 0
	br i1 %124, label %138, label %148

125:
	br label %75

126:
	%127 = load i32*, i32** %12
	%128 = bitcast i32* %127 to i32*
	%129 = load i32*, i32** %4
	%130 = bitcast i32* %129 to i32*
	%131 = load i32, i32* %15
	%132 = call i32* @rt_substr(i32* %130, i32 %131, i32 1)
	%133 = bitcast i32* %132 to i32*
	%134 = call i32* @rt_strcat(i32* %128, i32* %133)
	%135 = bitcast i32* %134 to i32*
	store i32* %135, i32** %12
	%136 = load i32, i32* %15
	%137 = add i32 %136, 1
	store i32 %137, i32* %15
	br label %125

138:
	%139 = load i32*, i32** %12
	%140 = bitcast i32* %139 to i32*
	%141 = load i32*, i32** %4
	%142 = bitcast i32* %141 to i32*
	%143 = load i32, i32* %101
	%144 = call i32* @rt_substr(i32* %142, i32 %143, i32 -1)
	%145 = bitcast i32* %144 to i32*
	%146 = call i32* @rt_strcat(i32* %140, i32* %145)
	%147 = bitcast i32* %146 to i32*
	store i32* %147, i32** %12
	store i32 1, i32* %16
	br label %148

148:
	br label %125

dead67:
	ret i32* null
}

define i32* @rt_case(i32* %0, i32 %1) {
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
	%8 = alloca i32*
	%9 = load i32, i32* %4
	%10 = add i32 %9, 1
	%11 = call i32* @rt_bump(i32 %10)
	%12 = bitcast i32* %11 to i32*
	store i32* %12, i32** %8
	%13 = alloca i32
	%14 = load i32, i32* %3
	%15 = icmp slt i32 %14, 2
	%16 = zext i1 %15 to i32
	store i32 %16, i32* %13
	%17 = alloca i32
	%18 = load i32, i32* %3
	%19 = icmp eq i32 %18, 1
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	br i1 %23, label %30, label %24

24:
	%25 = load i32, i32* %3
	%26 = icmp eq i32 %25, 3
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	%29 = zext i1 %28 to i32
	br label %30

30:
	%31 = phi i32 [ %22, %entry ], [ %29, %24 ]
	store i32 %31, i32* %17
	%32 = alloca i32
	store i32 0, i32* %32
	%33 = alloca i32
	store i32 1, i32* %33
	br label %34

34:
	%35 = load i32, i32* %32
	%36 = load i32, i32* %4
	%37 = icmp slt i32 %35, %36
	%38 = zext i1 %37 to i32
	%39 = icmp ne i32 %38, 0
	br i1 %39, label %40, label %53

40:
	%41 = alloca i32
	%42 = load i32, i32* %32
	%43 = load i32*, i32** %2
	%44 = getelementptr i8, i32* %43, i32 %42
	%45 = load i8, i8* %44
	%46 = sext i8 %45 to i32
	%47 = and i32 %46, 255
	store i32 %47, i32* %41
	%48 = alloca i32
	%49 = load i32, i32* %17
	%50 = icmp ne i32 %49, 0
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %70, label %64

53:
	%54 = load i32, i32* %4
	%55 = load i32*, i32** %8
	%56 = getelementptr i8, i32* %55, i32 %54
	%57 = shl i32 0, 24
	%58 = ashr i32 %57, 24
	%59 = shl i32 %58, 24
	%60 = ashr i32 %59, 24
	%61 = trunc i32 %60 to i8
	store i8 %61, i8* %56
	%62 = load i32*, i32** %8
	%63 = bitcast i32* %62 to i32*
	ret i32* %63

64:
	%65 = load i32, i32* %33
	%66 = icmp ne i32 %65, 0
	%67 = zext i1 %66 to i32
	%68 = icmp ne i32 %67, 0
	%69 = zext i1 %68 to i32
	br label %70

70:
	%71 = phi i32 [ %51, %40 ], [ %69, %64 ]
	store i32 %71, i32* %48
	%72 = alloca i32
	%73 = load i32, i32* %41
	%74 = icmp slt i32 %73, 128
	%75 = zext i1 %74 to i32
	%76 = icmp ne i32 %75, 0
	br i1 %76, label %77, label %89

77:
	%78 = alloca i32
	%79 = load i32, i32* %41
	%80 = icmp sge i32 %79, 97
	%81 = zext i1 %80 to i32
	%82 = icmp ne i32 %81, 0
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %96, label %102

85:
	%86 = load i32, i32* %32
	%87 = load i32, i32* %72
	%88 = add i32 %86, %87
	store i32 %88, i32* %32
	store i32 0, i32* %33
	br label %34

89:
	%90 = load i32, i32* %41
	%91 = icmp eq i32 %90, 195
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	%94 = zext i1 %93 to i32
	%95 = icmp ne i32 %94, 0
	br i1 %95, label %166, label %174

96:
	%97 = load i32, i32* %41
	%98 = icmp sle i32 %97, 122
	%99 = zext i1 %98 to i32
	%100 = icmp ne i32 %99, 0
	%101 = zext i1 %100 to i32
	br label %102

102:
	%103 = phi i32 [ %83, %77 ], [ %101, %96 ]
	store i32 %103, i32* %78
	%104 = alloca i32
	%105 = load i32, i32* %41
	%106 = icmp sge i32 %105, 65
	%107 = zext i1 %106 to i32
	%108 = icmp ne i32 %107, 0
	%109 = zext i1 %108 to i32
	%110 = icmp ne i32 %109, 0
	br i1 %110, label %111, label %117

111:
	%112 = load i32, i32* %41
	%113 = icmp sle i32 %112, 90
	%114 = zext i1 %113 to i32
	%115 = icmp ne i32 %114, 0
	%116 = zext i1 %115 to i32
	br label %117

117:
	%118 = phi i32 [ %109, %102 ], [ %116, %111 ]
	store i32 %118, i32* %104
	%119 = alloca i32
	%120 = load i32, i32* %78
	%121 = icmp ne i32 %120, 0
	br i1 %121, label %122, label %125

122:
	%123 = load i32, i32* %41
	%124 = sub i32 %123, 32
	br label %127

125:
	%126 = load i32, i32* %41
	br label %127

127:
	%128 = phi i32 [ %124, %122 ], [ %126, %125 ]
	store i32 %128, i32* %119
	%129 = alloca i32
	%130 = load i32, i32* %104
	%131 = icmp ne i32 %130, 0
	br i1 %131, label %132, label %135

132:
	%133 = load i32, i32* %41
	%134 = add i32 %133, 32
	br label %137

135:
	%136 = load i32, i32* %41
	br label %137

137:
	%138 = phi i32 [ %134, %132 ], [ %136, %135 ]
	store i32 %138, i32* %129
	%139 = alloca i32
	%140 = load i32, i32* %13
	%141 = icmp ne i32 %140, 0
	br i1 %141, label %142, label %144

142:
	%143 = load i32, i32* %119
	br label %146

144:
	%145 = load i32, i32* %129
	br label %146

146:
	%147 = phi i32 [ %143, %142 ], [ %145, %144 ]
	store i32 %147, i32* %139
	%148 = load i32, i32* %32
	%149 = load i32*, i32** %8
	%150 = getelementptr i8, i32* %149, i32 %148
	%151 = load i32, i32* %48
	%152 = icmp ne i32 %151, 0
	br i1 %152, label %153, label %155

153:
	%154 = load i32, i32* %139
	br label %157

155:
	%156 = load i32, i32* %41
	br label %157

157:
	%158 = phi i32 [ %154, %153 ], [ %156, %155 ]
	%159 = shl i32 %158, 24
	%160 = ashr i32 %159, 24
	%161 = shl i32 %160, 24
	%162 = ashr i32 %161, 24
	%163 = shl i32 %162, 24
	%164 = ashr i32 %163, 24
	%165 = trunc i32 %164 to i8
	store i8 %165, i8* %150
	store i32 1, i32* %72
	br label %85

166:
	%167 = load i32, i32* %32
	%168 = add i32 %167, 1
	%169 = load i32, i32* %4
	%170 = icmp slt i32 %168, %169
	%171 = zext i1 %170 to i32
	%172 = icmp ne i32 %171, 0
	%173 = zext i1 %172 to i32
	br label %174

174:
	%175 = phi i32 [ %94, %89 ], [ %173, %166 ]
	%176 = icmp ne i32 %175, 0
	br i1 %176, label %177, label %194

177:
	%178 = alloca i32
	%179 = load i32, i32* %32
	%180 = add i32 %179, 1
	%181 = load i32*, i32** %2
	%182 = getelementptr i8, i32* %181, i32 %180
	%183 = load i8, i8* %182
	%184 = sext i8 %183 to i32
	%185 = and i32 %184, 255
	store i32 %185, i32* %178
	%186 = alloca i32
	%187 = load i32, i32* %178
	%188 = icmp sge i32 %187, 128
	%189 = zext i1 %188 to i32
	%190 = icmp ne i32 %189, 0
	%191 = zext i1 %190 to i32
	%192 = icmp ne i32 %191, 0
	br i1 %192, label %200, label %206

193:
	br label %85

194:
	%195 = alloca i32
	store i32 1, i32* %195
	%196 = load i32, i32* %41
	%197 = icmp sge i32 %196, 240
	%198 = zext i1 %197 to i32
	%199 = icmp ne i32 %198, 0
	br i1 %199, label %311, label %320

200:
	%201 = load i32, i32* %178
	%202 = icmp sle i32 %201, 158
	%203 = zext i1 %202 to i32
	%204 = icmp ne i32 %203, 0
	%205 = zext i1 %204 to i32
	br label %206

206:
	%207 = phi i32 [ %191, %177 ], [ %205, %200 ]
	%208 = icmp ne i32 %207, 0
	br i1 %208, label %209, label %215

209:
	%210 = load i32, i32* %178
	%211 = icmp ne i32 %210, 151
	%212 = zext i1 %211 to i32
	%213 = icmp ne i32 %212, 0
	%214 = zext i1 %213 to i32
	br label %215

215:
	%216 = phi i32 [ %207, %206 ], [ %214, %209 ]
	store i32 %216, i32* %186
	%217 = alloca i32
	%218 = load i32, i32* %178
	%219 = icmp sge i32 %218, 160
	%220 = zext i1 %219 to i32
	%221 = icmp ne i32 %220, 0
	%222 = zext i1 %221 to i32
	%223 = icmp ne i32 %222, 0
	br i1 %223, label %224, label %230

224:
	%225 = load i32, i32* %178
	%226 = icmp sle i32 %225, 190
	%227 = zext i1 %226 to i32
	%228 = icmp ne i32 %227, 0
	%229 = zext i1 %228 to i32
	br label %230

230:
	%231 = phi i32 [ %222, %215 ], [ %229, %224 ]
	%232 = icmp ne i32 %231, 0
	br i1 %232, label %233, label %239

233:
	%234 = load i32, i32* %178
	%235 = icmp ne i32 %234, 183
	%236 = zext i1 %235 to i32
	%237 = icmp ne i32 %236, 0
	%238 = zext i1 %237 to i32
	br label %239

239:
	%240 = phi i32 [ %231, %230 ], [ %238, %233 ]
	store i32 %240, i32* %217
	%241 = alloca i32
	%242 = load i32, i32* %178
	store i32 %242, i32* %241
	%243 = load i32, i32* %13
	%244 = icmp ne i32 %243, 0
	%245 = zext i1 %244 to i32
	%246 = icmp ne i32 %245, 0
	%247 = zext i1 %246 to i32
	%248 = icmp ne i32 %247, 0
	br i1 %248, label %249, label %255

249:
	%250 = load i32, i32* %217
	%251 = icmp ne i32 %250, 0
	%252 = zext i1 %251 to i32
	%253 = icmp ne i32 %252, 0
	%254 = zext i1 %253 to i32
	br label %255

255:
	%256 = phi i32 [ %247, %239 ], [ %254, %249 ]
	%257 = icmp ne i32 %256, 0
	br i1 %257, label %258, label %261

258:
	%259 = load i32, i32* %178
	%260 = sub i32 %259, 32
	store i32 %260, i32* %241
	br label %261

261:
	%262 = load i32, i32* %13
	%263 = icmp eq i32 %262, 0
	%264 = zext i1 %263 to i32
	%265 = icmp ne i32 %264, 0
	%266 = zext i1 %265 to i32
	%267 = icmp ne i32 %266, 0
	br i1 %267, label %268, label %274

268:
	%269 = load i32, i32* %186
	%270 = icmp ne i32 %269, 0
	%271 = zext i1 %270 to i32
	%272 = icmp ne i32 %271, 0
	%273 = zext i1 %272 to i32
	br label %274

274:
	%275 = phi i32 [ %266, %261 ], [ %273, %268 ]
	%276 = icmp ne i32 %275, 0
	br i1 %276, label %277, label %280

277:
	%278 = load i32, i32* %178
	%279 = add i32 %278, 32
	store i32 %279, i32* %241
	br label %280

280:
	%281 = load i32, i32* %32
	%282 = load i32*, i32** %8
	%283 = getelementptr i8, i32* %282, i32 %281
	%284 = load i32, i32* %41
	%285 = shl i32 %284, 24
	%286 = ashr i32 %285, 24
	%287 = shl i32 %286, 24
	%288 = ashr i32 %287, 24
	%289 = shl i32 %288, 24
	%290 = ashr i32 %289, 24
	%291 = trunc i32 %290 to i8
	store i8 %291, i8* %283
	%292 = load i32, i32* %32
	%293 = add i32 %292, 1
	%294 = load i32*, i32** %8
	%295 = getelementptr i8, i32* %294, i32 %293
	%296 = load i32, i32* %48
	%297 = icmp ne i32 %296, 0
	br i1 %297, label %298, label %300

298:
	%299 = load i32, i32* %241
	br label %302

300:
	%301 = load i32, i32* %178
	br label %302

302:
	%303 = phi i32 [ %299, %298 ], [ %301, %300 ]
	%304 = shl i32 %303, 24
	%305 = ashr i32 %304, 24
	%306 = shl i32 %305, 24
	%307 = ashr i32 %306, 24
	%308 = shl i32 %307, 24
	%309 = ashr i32 %308, 24
	%310 = trunc i32 %309 to i8
	store i8 %310, i8* %295
	store i32 2, i32* %72
	br label %193

311:
	store i32 4, i32* %195
	br label %312

312:
	%313 = load i32, i32* %32
	%314 = load i32, i32* %195
	%315 = add i32 %313, %314
	%316 = load i32, i32* %4
	%317 = icmp sgt i32 %315, %316
	%318 = zext i1 %317 to i32
	%319 = icmp ne i32 %318, 0
	br i1 %319, label %334, label %338

320:
	%321 = load i32, i32* %41
	%322 = icmp sge i32 %321, 224
	%323 = zext i1 %322 to i32
	%324 = icmp ne i32 %323, 0
	br i1 %324, label %325, label %327

325:
	store i32 3, i32* %195
	br label %326

326:
	br label %312

327:
	%328 = load i32, i32* %41
	%329 = icmp sge i32 %328, 192
	%330 = zext i1 %329 to i32
	%331 = icmp ne i32 %330, 0
	br i1 %331, label %332, label %333

332:
	store i32 2, i32* %195
	br label %333

333:
	br label %326

334:
	%335 = load i32, i32* %4
	%336 = load i32, i32* %32
	%337 = sub i32 %335, %336
	store i32 %337, i32* %195
	br label %338

338:
	%339 = alloca i32
	store i32 0, i32* %339
	br label %340

340:
	%341 = load i32, i32* %339
	%342 = load i32, i32* %195
	%343 = icmp slt i32 %341, %342
	%344 = zext i1 %343 to i32
	%345 = icmp ne i32 %344, 0
	br i1 %345, label %346, label %366

346:
	%347 = load i32, i32* %32
	%348 = load i32, i32* %339
	%349 = add i32 %347, %348
	%350 = load i32*, i32** %8
	%351 = getelementptr i8, i32* %350, i32 %349
	%352 = load i32, i32* %32
	%353 = load i32, i32* %339
	%354 = add i32 %352, %353
	%355 = load i32*, i32** %2
	%356 = getelementptr i8, i32* %355, i32 %354
	%357 = load i8, i8* %356
	%358 = sext i8 %357 to i32
	%359 = shl i32 %358, 24
	%360 = ashr i32 %359, 24
	%361 = shl i32 %360, 24
	%362 = ashr i32 %361, 24
	%363 = trunc i32 %362 to i8
	store i8 %363, i8* %351
	%364 = load i32, i32* %339
	%365 = add i32 %364, 1
	store i32 %365, i32* %339
	br label %340

366:
	%367 = load i32, i32* %195
	store i32 %367, i32* %72
	br label %193

dead68:
	ret i32* null
}

define i32* @rt_shquote(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32*
	%7 = load i32, i32* %2
	%8 = mul i32 %7, 4
	%9 = add i32 %8, 8
	%10 = call i32* @rt_bump(i32 %9)
	%11 = bitcast i32* %10 to i32*
	store i32* %11, i32** %6
	%12 = alloca i32
	store i32 0, i32* %12
	%13 = alloca i32
	store i32 0, i32* %13
	%14 = alloca i32
	store i32 0, i32* %14
	br label %15

15:
	%16 = load i32, i32* %13
	%17 = load i32, i32* %2
	%18 = icmp slt i32 %16, %17
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %36

21:
	%22 = alloca i32
	%23 = load i32, i32* %13
	%24 = load i32*, i32** %1
	%25 = getelementptr i8, i32* %24, i32 %23
	%26 = load i8, i8* %25
	%27 = sext i8 %26 to i32
	%28 = and i32 %27, 255
	store i32 %28, i32* %22
	%29 = alloca i32
	%30 = load i32, i32* %22
	%31 = icmp slt i32 %30, 32
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %47, label %41

36:
	%37 = load i32, i32* %14
	%38 = icmp ne i32 %37, 0
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	br i1 %40, label %58, label %91

41:
	%42 = load i32, i32* %22
	%43 = icmp eq i32 %42, 127
	%44 = zext i1 %43 to i32
	%45 = icmp ne i32 %44, 0
	%46 = zext i1 %45 to i32
	br label %47

47:
	%48 = phi i32 [ %34, %21 ], [ %46, %41 ]
	store i32 %48, i32* %29
	%49 = load i32, i32* %29
	%50 = icmp ne i32 %49, 0
	br i1 %50, label %51, label %52

51:
	br label %54

52:
	%53 = load i32, i32* %14
	br label %54

54:
	%55 = phi i32 [ 1, %51 ], [ %53, %52 ]
	store i32 %55, i32* %14
	%56 = load i32, i32* %13
	%57 = add i32 %56, 1
	store i32 %57, i32* %13
	br label %15

58:
	%59 = load i32, i32* %12
	%60 = load i32*, i32** %6
	%61 = getelementptr i8, i32* %60, i32 %59
	%62 = shl i32 36, 24
	%63 = ashr i32 %62, 24
	%64 = shl i32 %63, 24
	%65 = ashr i32 %64, 24
	%66 = trunc i32 %65 to i8
	store i8 %66, i8* %61
	%67 = load i32, i32* %12
	%68 = add i32 %67, 1
	store i32 %68, i32* %12
	%69 = load i32, i32* %12
	%70 = load i32*, i32** %6
	%71 = getelementptr i8, i32* %70, i32 %69
	%72 = shl i32 39, 24
	%73 = ashr i32 %72, 24
	%74 = shl i32 %73, 24
	%75 = ashr i32 %74, 24
	%76 = trunc i32 %75 to i8
	store i8 %76, i8* %71
	%77 = load i32, i32* %12
	%78 = add i32 %77, 1
	store i32 %78, i32* %12
	%79 = alloca i32
	store i32 0, i32* %79
	br label %103

80:
	%81 = load i32, i32* %12
	%82 = load i32*, i32** %6
	%83 = getelementptr i8, i32* %82, i32 %81
	%84 = shl i32 0, 24
	%85 = ashr i32 %84, 24
	%86 = shl i32 %85, 24
	%87 = ashr i32 %86, 24
	%88 = trunc i32 %87 to i8
	store i8 %88, i8* %83
	%89 = load i32*, i32** %6
	%90 = bitcast i32* %89 to i32*
	ret i32* %90

91:
	%92 = load i32, i32* %12
	%93 = load i32*, i32** %6
	%94 = getelementptr i8, i32* %93, i32 %92
	%95 = shl i32 39, 24
	%96 = ashr i32 %95, 24
	%97 = shl i32 %96, 24
	%98 = ashr i32 %97, 24
	%99 = trunc i32 %98 to i8
	store i8 %99, i8* %94
	%100 = load i32, i32* %12
	%101 = add i32 %100, 1
	store i32 %101, i32* %12
	%102 = alloca i32
	store i32 0, i32* %102
	br label %460

103:
	%104 = load i32, i32* %79
	%105 = load i32, i32* %2
	%106 = icmp slt i32 %104, %105
	%107 = zext i1 %106 to i32
	%108 = icmp ne i32 %107, 0
	br i1 %108, label %109, label %121

109:
	%110 = alloca i32
	%111 = load i32, i32* %79
	%112 = load i32*, i32** %1
	%113 = getelementptr i8, i32* %112, i32 %111
	%114 = load i8, i8* %113
	%115 = sext i8 %114 to i32
	%116 = and i32 %115, 255
	store i32 %116, i32* %110
	%117 = load i32, i32* %110
	%118 = icmp eq i32 %117, 39
	%119 = zext i1 %118 to i32
	%120 = icmp ne i32 %119, 0
	br i1 %120, label %132, label %156

121:
	%122 = load i32, i32* %12
	%123 = load i32*, i32** %6
	%124 = getelementptr i8, i32* %123, i32 %122
	%125 = shl i32 39, 24
	%126 = ashr i32 %125, 24
	%127 = shl i32 %126, 24
	%128 = ashr i32 %127, 24
	%129 = trunc i32 %128 to i8
	store i8 %129, i8* %124
	%130 = load i32, i32* %12
	%131 = add i32 %130, 1
	store i32 %131, i32* %12
	br label %80

132:
	%133 = load i32, i32* %12
	%134 = load i32*, i32** %6
	%135 = getelementptr i8, i32* %134, i32 %133
	%136 = shl i32 92, 24
	%137 = ashr i32 %136, 24
	%138 = shl i32 %137, 24
	%139 = ashr i32 %138, 24
	%140 = trunc i32 %139 to i8
	store i8 %140, i8* %135
	%141 = load i32, i32* %12
	%142 = add i32 %141, 1
	store i32 %142, i32* %12
	%143 = load i32, i32* %12
	%144 = load i32*, i32** %6
	%145 = getelementptr i8, i32* %144, i32 %143
	%146 = shl i32 39, 24
	%147 = ashr i32 %146, 24
	%148 = shl i32 %147, 24
	%149 = ashr i32 %148, 24
	%150 = trunc i32 %149 to i8
	store i8 %150, i8* %145
	%151 = load i32, i32* %12
	%152 = add i32 %151, 1
	store i32 %152, i32* %12
	br label %153

153:
	%154 = load i32, i32* %79
	%155 = add i32 %154, 1
	store i32 %155, i32* %79
	br label %103

156:
	%157 = load i32, i32* %110
	%158 = icmp eq i32 %157, 92
	%159 = zext i1 %158 to i32
	%160 = icmp ne i32 %159, 0
	br i1 %160, label %161, label %183

161:
	%162 = load i32, i32* %12
	%163 = load i32*, i32** %6
	%164 = getelementptr i8, i32* %163, i32 %162
	%165 = shl i32 92, 24
	%166 = ashr i32 %165, 24
	%167 = shl i32 %166, 24
	%168 = ashr i32 %167, 24
	%169 = trunc i32 %168 to i8
	store i8 %169, i8* %164
	%170 = load i32, i32* %12
	%171 = add i32 %170, 1
	store i32 %171, i32* %12
	%172 = load i32, i32* %12
	%173 = load i32*, i32** %6
	%174 = getelementptr i8, i32* %173, i32 %172
	%175 = shl i32 92, 24
	%176 = ashr i32 %175, 24
	%177 = shl i32 %176, 24
	%178 = ashr i32 %177, 24
	%179 = trunc i32 %178 to i8
	store i8 %179, i8* %174
	%180 = load i32, i32* %12
	%181 = add i32 %180, 1
	store i32 %181, i32* %12
	br label %182

182:
	br label %153

183:
	%184 = load i32, i32* %110
	%185 = icmp eq i32 %184, 7
	%186 = zext i1 %185 to i32
	%187 = icmp ne i32 %186, 0
	br i1 %187, label %188, label %210

188:
	%189 = load i32, i32* %12
	%190 = load i32*, i32** %6
	%191 = getelementptr i8, i32* %190, i32 %189
	%192 = shl i32 92, 24
	%193 = ashr i32 %192, 24
	%194 = shl i32 %193, 24
	%195 = ashr i32 %194, 24
	%196 = trunc i32 %195 to i8
	store i8 %196, i8* %191
	%197 = load i32, i32* %12
	%198 = add i32 %197, 1
	store i32 %198, i32* %12
	%199 = load i32, i32* %12
	%200 = load i32*, i32** %6
	%201 = getelementptr i8, i32* %200, i32 %199
	%202 = shl i32 97, 24
	%203 = ashr i32 %202, 24
	%204 = shl i32 %203, 24
	%205 = ashr i32 %204, 24
	%206 = trunc i32 %205 to i8
	store i8 %206, i8* %201
	%207 = load i32, i32* %12
	%208 = add i32 %207, 1
	store i32 %208, i32* %12
	br label %209

209:
	br label %182

210:
	%211 = load i32, i32* %110
	%212 = icmp eq i32 %211, 8
	%213 = zext i1 %212 to i32
	%214 = icmp ne i32 %213, 0
	br i1 %214, label %215, label %237

215:
	%216 = load i32, i32* %12
	%217 = load i32*, i32** %6
	%218 = getelementptr i8, i32* %217, i32 %216
	%219 = shl i32 92, 24
	%220 = ashr i32 %219, 24
	%221 = shl i32 %220, 24
	%222 = ashr i32 %221, 24
	%223 = trunc i32 %222 to i8
	store i8 %223, i8* %218
	%224 = load i32, i32* %12
	%225 = add i32 %224, 1
	store i32 %225, i32* %12
	%226 = load i32, i32* %12
	%227 = load i32*, i32** %6
	%228 = getelementptr i8, i32* %227, i32 %226
	%229 = shl i32 98, 24
	%230 = ashr i32 %229, 24
	%231 = shl i32 %230, 24
	%232 = ashr i32 %231, 24
	%233 = trunc i32 %232 to i8
	store i8 %233, i8* %228
	%234 = load i32, i32* %12
	%235 = add i32 %234, 1
	store i32 %235, i32* %12
	br label %236

236:
	br label %209

237:
	%238 = load i32, i32* %110
	%239 = icmp eq i32 %238, 9
	%240 = zext i1 %239 to i32
	%241 = icmp ne i32 %240, 0
	br i1 %241, label %242, label %264

242:
	%243 = load i32, i32* %12
	%244 = load i32*, i32** %6
	%245 = getelementptr i8, i32* %244, i32 %243
	%246 = shl i32 92, 24
	%247 = ashr i32 %246, 24
	%248 = shl i32 %247, 24
	%249 = ashr i32 %248, 24
	%250 = trunc i32 %249 to i8
	store i8 %250, i8* %245
	%251 = load i32, i32* %12
	%252 = add i32 %251, 1
	store i32 %252, i32* %12
	%253 = load i32, i32* %12
	%254 = load i32*, i32** %6
	%255 = getelementptr i8, i32* %254, i32 %253
	%256 = shl i32 116, 24
	%257 = ashr i32 %256, 24
	%258 = shl i32 %257, 24
	%259 = ashr i32 %258, 24
	%260 = trunc i32 %259 to i8
	store i8 %260, i8* %255
	%261 = load i32, i32* %12
	%262 = add i32 %261, 1
	store i32 %262, i32* %12
	br label %263

263:
	br label %236

264:
	%265 = load i32, i32* %110
	%266 = icmp eq i32 %265, 10
	%267 = zext i1 %266 to i32
	%268 = icmp ne i32 %267, 0
	br i1 %268, label %269, label %291

269:
	%270 = load i32, i32* %12
	%271 = load i32*, i32** %6
	%272 = getelementptr i8, i32* %271, i32 %270
	%273 = shl i32 92, 24
	%274 = ashr i32 %273, 24
	%275 = shl i32 %274, 24
	%276 = ashr i32 %275, 24
	%277 = trunc i32 %276 to i8
	store i8 %277, i8* %272
	%278 = load i32, i32* %12
	%279 = add i32 %278, 1
	store i32 %279, i32* %12
	%280 = load i32, i32* %12
	%281 = load i32*, i32** %6
	%282 = getelementptr i8, i32* %281, i32 %280
	%283 = shl i32 110, 24
	%284 = ashr i32 %283, 24
	%285 = shl i32 %284, 24
	%286 = ashr i32 %285, 24
	%287 = trunc i32 %286 to i8
	store i8 %287, i8* %282
	%288 = load i32, i32* %12
	%289 = add i32 %288, 1
	store i32 %289, i32* %12
	br label %290

290:
	br label %263

291:
	%292 = load i32, i32* %110
	%293 = icmp eq i32 %292, 11
	%294 = zext i1 %293 to i32
	%295 = icmp ne i32 %294, 0
	br i1 %295, label %296, label %318

296:
	%297 = load i32, i32* %12
	%298 = load i32*, i32** %6
	%299 = getelementptr i8, i32* %298, i32 %297
	%300 = shl i32 92, 24
	%301 = ashr i32 %300, 24
	%302 = shl i32 %301, 24
	%303 = ashr i32 %302, 24
	%304 = trunc i32 %303 to i8
	store i8 %304, i8* %299
	%305 = load i32, i32* %12
	%306 = add i32 %305, 1
	store i32 %306, i32* %12
	%307 = load i32, i32* %12
	%308 = load i32*, i32** %6
	%309 = getelementptr i8, i32* %308, i32 %307
	%310 = shl i32 118, 24
	%311 = ashr i32 %310, 24
	%312 = shl i32 %311, 24
	%313 = ashr i32 %312, 24
	%314 = trunc i32 %313 to i8
	store i8 %314, i8* %309
	%315 = load i32, i32* %12
	%316 = add i32 %315, 1
	store i32 %316, i32* %12
	br label %317

317:
	br label %290

318:
	%319 = load i32, i32* %110
	%320 = icmp eq i32 %319, 12
	%321 = zext i1 %320 to i32
	%322 = icmp ne i32 %321, 0
	br i1 %322, label %323, label %345

323:
	%324 = load i32, i32* %12
	%325 = load i32*, i32** %6
	%326 = getelementptr i8, i32* %325, i32 %324
	%327 = shl i32 92, 24
	%328 = ashr i32 %327, 24
	%329 = shl i32 %328, 24
	%330 = ashr i32 %329, 24
	%331 = trunc i32 %330 to i8
	store i8 %331, i8* %326
	%332 = load i32, i32* %12
	%333 = add i32 %332, 1
	store i32 %333, i32* %12
	%334 = load i32, i32* %12
	%335 = load i32*, i32** %6
	%336 = getelementptr i8, i32* %335, i32 %334
	%337 = shl i32 102, 24
	%338 = ashr i32 %337, 24
	%339 = shl i32 %338, 24
	%340 = ashr i32 %339, 24
	%341 = trunc i32 %340 to i8
	store i8 %341, i8* %336
	%342 = load i32, i32* %12
	%343 = add i32 %342, 1
	store i32 %343, i32* %12
	br label %344

344:
	br label %317

345:
	%346 = load i32, i32* %110
	%347 = icmp eq i32 %346, 13
	%348 = zext i1 %347 to i32
	%349 = icmp ne i32 %348, 0
	br i1 %349, label %350, label %372

350:
	%351 = load i32, i32* %12
	%352 = load i32*, i32** %6
	%353 = getelementptr i8, i32* %352, i32 %351
	%354 = shl i32 92, 24
	%355 = ashr i32 %354, 24
	%356 = shl i32 %355, 24
	%357 = ashr i32 %356, 24
	%358 = trunc i32 %357 to i8
	store i8 %358, i8* %353
	%359 = load i32, i32* %12
	%360 = add i32 %359, 1
	store i32 %360, i32* %12
	%361 = load i32, i32* %12
	%362 = load i32*, i32** %6
	%363 = getelementptr i8, i32* %362, i32 %361
	%364 = shl i32 114, 24
	%365 = ashr i32 %364, 24
	%366 = shl i32 %365, 24
	%367 = ashr i32 %366, 24
	%368 = trunc i32 %367 to i8
	store i8 %368, i8* %363
	%369 = load i32, i32* %12
	%370 = add i32 %369, 1
	store i32 %370, i32* %12
	br label %371

371:
	br label %344

372:
	%373 = load i32, i32* %110
	%374 = icmp slt i32 %373, 32
	%375 = zext i1 %374 to i32
	%376 = icmp ne i32 %375, 0
	%377 = zext i1 %376 to i32
	%378 = icmp ne i32 %377, 0
	br i1 %378, label %385, label %379

379:
	%380 = load i32, i32* %110
	%381 = icmp eq i32 %380, 127
	%382 = zext i1 %381 to i32
	%383 = icmp ne i32 %382, 0
	%384 = zext i1 %383 to i32
	br label %385

385:
	%386 = phi i32 [ %377, %372 ], [ %384, %379 ]
	%387 = icmp ne i32 %386, 0
	br i1 %387, label %388, label %446

388:
	%389 = load i32, i32* %12
	%390 = load i32*, i32** %6
	%391 = getelementptr i8, i32* %390, i32 %389
	%392 = shl i32 92, 24
	%393 = ashr i32 %392, 24
	%394 = shl i32 %393, 24
	%395 = ashr i32 %394, 24
	%396 = trunc i32 %395 to i8
	store i8 %396, i8* %391
	%397 = load i32, i32* %12
	%398 = add i32 %397, 1
	store i32 %398, i32* %12
	%399 = load i32, i32* %12
	%400 = load i32*, i32** %6
	%401 = getelementptr i8, i32* %400, i32 %399
	%402 = load i32, i32* %110
	%403 = sdiv i32 %402, 64
	%404 = add i32 48, %403
	%405 = shl i32 %404, 24
	%406 = ashr i32 %405, 24
	%407 = shl i32 %406, 24
	%408 = ashr i32 %407, 24
	%409 = shl i32 %408, 24
	%410 = ashr i32 %409, 24
	%411 = trunc i32 %410 to i8
	store i8 %411, i8* %401
	%412 = load i32, i32* %12
	%413 = add i32 %412, 1
	store i32 %413, i32* %12
	%414 = load i32, i32* %12
	%415 = load i32*, i32** %6
	%416 = getelementptr i8, i32* %415, i32 %414
	%417 = load i32, i32* %110
	%418 = sdiv i32 %417, 8
	%419 = srem i32 %418, 8
	%420 = add i32 48, %419
	%421 = shl i32 %420, 24
	%422 = ashr i32 %421, 24
	%423 = shl i32 %422, 24
	%424 = ashr i32 %423, 24
	%425 = shl i32 %424, 24
	%426 = ashr i32 %425, 24
	%427 = trunc i32 %426 to i8
	store i8 %427, i8* %416
	%428 = load i32, i32* %12
	%429 = add i32 %428, 1
	store i32 %429, i32* %12
	%430 = load i32, i32* %12
	%431 = load i32*, i32** %6
	%432 = getelementptr i8, i32* %431, i32 %430
	%433 = load i32, i32* %110
	%434 = srem i32 %433, 8
	%435 = add i32 48, %434
	%436 = shl i32 %435, 24
	%437 = ashr i32 %436, 24
	%438 = shl i32 %437, 24
	%439 = ashr i32 %438, 24
	%440 = shl i32 %439, 24
	%441 = ashr i32 %440, 24
	%442 = trunc i32 %441 to i8
	store i8 %442, i8* %432
	%443 = load i32, i32* %12
	%444 = add i32 %443, 1
	store i32 %444, i32* %12
	br label %445

445:
	br label %371

446:
	%447 = load i32, i32* %12
	%448 = load i32*, i32** %6
	%449 = getelementptr i8, i32* %448, i32 %447
	%450 = load i32, i32* %110
	%451 = shl i32 %450, 24
	%452 = ashr i32 %451, 24
	%453 = shl i32 %452, 24
	%454 = ashr i32 %453, 24
	%455 = shl i32 %454, 24
	%456 = ashr i32 %455, 24
	%457 = trunc i32 %456 to i8
	store i8 %457, i8* %449
	%458 = load i32, i32* %12
	%459 = add i32 %458, 1
	store i32 %459, i32* %12
	br label %445

460:
	%461 = load i32, i32* %102
	%462 = load i32, i32* %2
	%463 = icmp slt i32 %461, %462
	%464 = zext i1 %463 to i32
	%465 = icmp ne i32 %464, 0
	br i1 %465, label %466, label %478

466:
	%467 = alloca i32
	%468 = load i32, i32* %102
	%469 = load i32*, i32** %1
	%470 = getelementptr i8, i32* %469, i32 %468
	%471 = load i8, i8* %470
	%472 = sext i8 %471 to i32
	%473 = and i32 %472, 255
	store i32 %473, i32* %467
	%474 = load i32, i32* %467
	%475 = icmp eq i32 %474, 39
	%476 = zext i1 %475 to i32
	%477 = icmp ne i32 %476, 0
	br i1 %477, label %489, label %533

478:
	%479 = load i32, i32* %12
	%480 = load i32*, i32** %6
	%481 = getelementptr i8, i32* %480, i32 %479
	%482 = shl i32 39, 24
	%483 = ashr i32 %482, 24
	%484 = shl i32 %483, 24
	%485 = ashr i32 %484, 24
	%486 = trunc i32 %485 to i8
	store i8 %486, i8* %481
	%487 = load i32, i32* %12
	%488 = add i32 %487, 1
	store i32 %488, i32* %12
	br label %80

489:
	%490 = load i32, i32* %12
	%491 = load i32*, i32** %6
	%492 = getelementptr i8, i32* %491, i32 %490
	%493 = shl i32 39, 24
	%494 = ashr i32 %493, 24
	%495 = shl i32 %494, 24
	%496 = ashr i32 %495, 24
	%497 = trunc i32 %496 to i8
	store i8 %497, i8* %492
	%498 = load i32, i32* %12
	%499 = add i32 %498, 1
	store i32 %499, i32* %12
	%500 = load i32, i32* %12
	%501 = load i32*, i32** %6
	%502 = getelementptr i8, i32* %501, i32 %500
	%503 = shl i32 92, 24
	%504 = ashr i32 %503, 24
	%505 = shl i32 %504, 24
	%506 = ashr i32 %505, 24
	%507 = trunc i32 %506 to i8
	store i8 %507, i8* %502
	%508 = load i32, i32* %12
	%509 = add i32 %508, 1
	store i32 %509, i32* %12
	%510 = load i32, i32* %12
	%511 = load i32*, i32** %6
	%512 = getelementptr i8, i32* %511, i32 %510
	%513 = shl i32 39, 24
	%514 = ashr i32 %513, 24
	%515 = shl i32 %514, 24
	%516 = ashr i32 %515, 24
	%517 = trunc i32 %516 to i8
	store i8 %517, i8* %512
	%518 = load i32, i32* %12
	%519 = add i32 %518, 1
	store i32 %519, i32* %12
	%520 = load i32, i32* %12
	%521 = load i32*, i32** %6
	%522 = getelementptr i8, i32* %521, i32 %520
	%523 = shl i32 39, 24
	%524 = ashr i32 %523, 24
	%525 = shl i32 %524, 24
	%526 = ashr i32 %525, 24
	%527 = trunc i32 %526 to i8
	store i8 %527, i8* %522
	%528 = load i32, i32* %12
	%529 = add i32 %528, 1
	store i32 %529, i32* %12
	br label %530

530:
	%531 = load i32, i32* %102
	%532 = add i32 %531, 1
	store i32 %532, i32* %102
	br label %460

533:
	%534 = load i32, i32* %12
	%535 = load i32*, i32** %6
	%536 = getelementptr i8, i32* %535, i32 %534
	%537 = load i32, i32* %467
	%538 = shl i32 %537, 24
	%539 = ashr i32 %538, 24
	%540 = shl i32 %539, 24
	%541 = ashr i32 %540, 24
	%542 = shl i32 %541, 24
	%543 = ashr i32 %542, 24
	%544 = trunc i32 %543 to i8
	store i8 %544, i8* %536
	%545 = load i32, i32* %12
	%546 = add i32 %545, 1
	store i32 %546, i32* %12
	br label %530

dead69:
	ret i32* null
}

define i32 @rt_hexval(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = load i32, i32* %1
	%3 = icmp sge i32 %2, 48
	%4 = zext i1 %3 to i32
	%5 = icmp ne i32 %4, 0
	%6 = zext i1 %5 to i32
	%7 = icmp ne i32 %6, 0
	br i1 %7, label %8, label %14

8:
	%9 = load i32, i32* %1
	%10 = icmp sle i32 %9, 57
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	%13 = zext i1 %12 to i32
	br label %14

14:
	%15 = phi i32 [ %6, %entry ], [ %13, %8 ]
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %20

17:
	%18 = load i32, i32* %1
	%19 = sub i32 %18, 48
	ret i32 %19

20:
	%21 = load i32, i32* %1
	%22 = icmp sge i32 %21, 97
	%23 = zext i1 %22 to i32
	%24 = icmp ne i32 %23, 0
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %27, label %33

dead70:
	br label %20

27:
	%28 = load i32, i32* %1
	%29 = icmp sle i32 %28, 102
	%30 = zext i1 %29 to i32
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	br label %33

33:
	%34 = phi i32 [ %25, %20 ], [ %32, %27 ]
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %39

36:
	%37 = load i32, i32* %1
	%38 = sub i32 %37, 87
	ret i32 %38

39:
	%40 = load i32, i32* %1
	%41 = icmp sge i32 %40, 65
	%42 = zext i1 %41 to i32
	%43 = icmp ne i32 %42, 0
	%44 = zext i1 %43 to i32
	%45 = icmp ne i32 %44, 0
	br i1 %45, label %46, label %52

dead71:
	br label %39

46:
	%47 = load i32, i32* %1
	%48 = icmp sle i32 %47, 70
	%49 = zext i1 %48 to i32
	%50 = icmp ne i32 %49, 0
	%51 = zext i1 %50 to i32
	br label %52

52:
	%53 = phi i32 [ %44, %39 ], [ %51, %46 ]
	%54 = icmp ne i32 %53, 0
	br i1 %54, label %55, label %58

55:
	%56 = load i32, i32* %1
	%57 = sub i32 %56, 55
	ret i32 %57

58:
	ret i32 -1

dead72:
	br label %58

dead73:
	ret i32 0
}

define i32* @rt_ansic(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32*
	%7 = load i32, i32* %2
	%8 = add i32 %7, 1
	%9 = call i32* @rt_bump(i32 %8)
	%10 = bitcast i32* %9 to i32*
	store i32* %10, i32** %6
	%11 = alloca i32
	store i32 0, i32* %11
	%12 = alloca i32
	store i32 0, i32* %12
	br label %13

13:
	%14 = load i32, i32* %11
	%15 = load i32, i32* %2
	%16 = icmp slt i32 %14, %15
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %19, label %34

19:
	%20 = alloca i32
	%21 = load i32, i32* %11
	%22 = load i32*, i32** %1
	%23 = getelementptr i8, i32* %22, i32 %21
	%24 = load i8, i8* %23
	%25 = sext i8 %24 to i32
	%26 = and i32 %25, 255
	store i32 %26, i32* %20
	%27 = alloca i32
	%28 = load i32, i32* %11
	%29 = add i32 %28, 1
	%30 = load i32, i32* %2
	%31 = icmp slt i32 %29, %30
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	br i1 %33, label %45, label %48

34:
	%35 = load i32, i32* %12
	%36 = load i32*, i32** %6
	%37 = getelementptr i8, i32* %36, i32 %35
	%38 = shl i32 0, 24
	%39 = ashr i32 %38, 24
	%40 = shl i32 %39, 24
	%41 = ashr i32 %40, 24
	%42 = trunc i32 %41 to i8
	store i8 %42, i8* %37
	%43 = load i32*, i32** %6
	%44 = bitcast i32* %43 to i32*
	ret i32* %44

45:
	%46 = load i32, i32* %11
	%47 = add i32 %46, 1
	br label %50

48:
	%49 = load i32, i32* %11
	br label %50

50:
	%51 = phi i32 [ %47, %45 ], [ %49, %48 ]
	%52 = load i32*, i32** %1
	%53 = getelementptr i8, i32* %52, i32 %51
	%54 = load i8, i8* %53
	%55 = sext i8 %54 to i32
	%56 = and i32 %55, 255
	store i32 %56, i32* %27
	%57 = load i32, i32* %20
	%58 = icmp eq i32 %57, 92
	%59 = zext i1 %58 to i32
	%60 = icmp ne i32 %59, 0
	%61 = zext i1 %60 to i32
	%62 = icmp ne i32 %61, 0
	br i1 %62, label %63, label %71

63:
	%64 = load i32, i32* %11
	%65 = add i32 %64, 1
	%66 = load i32, i32* %2
	%67 = icmp slt i32 %65, %66
	%68 = zext i1 %67 to i32
	%69 = icmp ne i32 %68, 0
	%70 = zext i1 %69 to i32
	br label %71

71:
	%72 = phi i32 [ %61, %50 ], [ %70, %63 ]
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %74, label %80

74:
	%75 = load i32, i32* %27
	%76 = icmp sge i32 %75, 48
	%77 = zext i1 %76 to i32
	%78 = icmp ne i32 %77, 0
	%79 = zext i1 %78 to i32
	br label %80

80:
	%81 = phi i32 [ %72, %71 ], [ %79, %74 ]
	%82 = icmp ne i32 %81, 0
	br i1 %82, label %83, label %89

83:
	%84 = load i32, i32* %27
	%85 = icmp sle i32 %84, 55
	%86 = zext i1 %85 to i32
	%87 = icmp ne i32 %86, 0
	%88 = zext i1 %87 to i32
	br label %89

89:
	%90 = phi i32 [ %81, %80 ], [ %88, %83 ]
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %92, label %98

92:
	%93 = alloca i32
	store i32 0, i32* %93
	%94 = alloca i32
	store i32 0, i32* %94
	%95 = load i32, i32* %11
	%96 = add i32 %95, 1
	store i32 %96, i32* %11
	br label %105

97:
	br label %13

98:
	%99 = load i32, i32* %20
	%100 = icmp eq i32 %99, 92
	%101 = zext i1 %100 to i32
	%102 = icmp ne i32 %101, 0
	%103 = zext i1 %102 to i32
	%104 = icmp ne i32 %103, 0
	br i1 %104, label %171, label %179

105:
	%106 = load i32, i32* %94
	%107 = icmp slt i32 %106, 3
	%108 = zext i1 %107 to i32
	%109 = icmp ne i32 %108, 0
	%110 = zext i1 %109 to i32
	%111 = icmp ne i32 %110, 0
	br i1 %111, label %141, label %148

112:
	%113 = alloca i32
	%114 = load i32, i32* %11
	%115 = load i32*, i32** %1
	%116 = getelementptr i8, i32* %115, i32 %114
	%117 = load i8, i8* %116
	%118 = sext i8 %117 to i32
	%119 = and i32 %118, 255
	store i32 %119, i32* %113
	%120 = load i32, i32* %113
	%121 = icmp slt i32 %120, 48
	%122 = zext i1 %121 to i32
	%123 = icmp ne i32 %122, 0
	%124 = zext i1 %123 to i32
	%125 = icmp ne i32 %124, 0
	br i1 %125, label %157, label %151

126:
	%127 = load i32, i32* %12
	%128 = load i32*, i32** %6
	%129 = getelementptr i8, i32* %128, i32 %127
	%130 = load i32, i32* %93
	%131 = srem i32 %130, 256
	%132 = shl i32 %131, 24
	%133 = ashr i32 %132, 24
	%134 = shl i32 %133, 24
	%135 = ashr i32 %134, 24
	%136 = shl i32 %135, 24
	%137 = ashr i32 %136, 24
	%138 = trunc i32 %137 to i8
	store i8 %138, i8* %129
	%139 = load i32, i32* %12
	%140 = add i32 %139, 1
	store i32 %140, i32* %12
	br label %97

141:
	%142 = load i32, i32* %11
	%143 = load i32, i32* %2
	%144 = icmp slt i32 %142, %143
	%145 = zext i1 %144 to i32
	%146 = icmp ne i32 %145, 0
	%147 = zext i1 %146 to i32
	br label %148

148:
	%149 = phi i32 [ %110, %105 ], [ %147, %141 ]
	%150 = icmp ne i32 %149, 0
	br i1 %150, label %112, label %126

151:
	%152 = load i32, i32* %113
	%153 = icmp sgt i32 %152, 55
	%154 = zext i1 %153 to i32
	%155 = icmp ne i32 %154, 0
	%156 = zext i1 %155 to i32
	br label %157

157:
	%158 = phi i32 [ %124, %112 ], [ %156, %151 ]
	%159 = icmp ne i32 %158, 0
	br i1 %159, label %160, label %161

160:
	br label %126

161:
	%162 = load i32, i32* %93
	%163 = mul i32 %162, 8
	%164 = load i32, i32* %113
	%165 = sub i32 %164, 48
	%166 = add i32 %163, %165
	store i32 %166, i32* %93
	%167 = load i32, i32* %11
	%168 = add i32 %167, 1
	store i32 %168, i32* %11
	%169 = load i32, i32* %94
	%170 = add i32 %169, 1
	store i32 %170, i32* %94
	br label %105

dead74:
	br label %161

171:
	%172 = load i32, i32* %11
	%173 = add i32 %172, 1
	%174 = load i32, i32* %2
	%175 = icmp slt i32 %173, %174
	%176 = zext i1 %175 to i32
	%177 = icmp ne i32 %176, 0
	%178 = zext i1 %177 to i32
	br label %179

179:
	%180 = phi i32 [ %103, %98 ], [ %178, %171 ]
	%181 = icmp ne i32 %180, 0
	br i1 %181, label %182, label %188

182:
	%183 = load i32, i32* %27
	%184 = icmp eq i32 %183, 120
	%185 = zext i1 %184 to i32
	%186 = icmp ne i32 %185, 0
	%187 = zext i1 %186 to i32
	br label %188

188:
	%189 = phi i32 [ %180, %179 ], [ %187, %182 ]
	%190 = icmp ne i32 %189, 0
	br i1 %190, label %191, label %198

191:
	%192 = load i32, i32* %11
	%193 = add i32 %192, 2
	%194 = load i32, i32* %2
	%195 = icmp slt i32 %193, %194
	%196 = zext i1 %195 to i32
	%197 = icmp ne i32 %196, 0
	br i1 %197, label %201, label %204

198:
	%199 = phi i32 [ %189, %188 ], [ %217, %206 ]
	%200 = icmp ne i32 %199, 0
	br i1 %200, label %218, label %226

201:
	%202 = load i32, i32* %11
	%203 = add i32 %202, 2
	br label %206

204:
	%205 = load i32, i32* %11
	br label %206

206:
	%207 = phi i32 [ %203, %201 ], [ %205, %204 ]
	%208 = load i32*, i32** %1
	%209 = getelementptr i8, i32* %208, i32 %207
	%210 = load i8, i8* %209
	%211 = sext i8 %210 to i32
	%212 = and i32 %211, 255
	%213 = call i32 @rt_hexval(i32 %212)
	%214 = icmp sge i32 %213, 0
	%215 = zext i1 %214 to i32
	%216 = icmp ne i32 %215, 0
	%217 = zext i1 %216 to i32
	br label %198

218:
	%219 = load i32, i32* %11
	%220 = add i32 %219, 2
	%221 = load i32, i32* %2
	%222 = icmp slt i32 %220, %221
	%223 = zext i1 %222 to i32
	%224 = icmp ne i32 %223, 0
	%225 = zext i1 %224 to i32
	br label %226

226:
	%227 = phi i32 [ %199, %198 ], [ %225, %218 ]
	%228 = icmp ne i32 %227, 0
	br i1 %228, label %229, label %235

229:
	%230 = alloca i32
	store i32 0, i32* %230
	%231 = alloca i32
	store i32 0, i32* %231
	%232 = load i32, i32* %11
	%233 = add i32 %232, 2
	store i32 %233, i32* %11
	br label %242

234:
	br label %97

235:
	%236 = load i32, i32* %20
	%237 = icmp eq i32 %236, 92
	%238 = zext i1 %237 to i32
	%239 = icmp ne i32 %238, 0
	%240 = zext i1 %239 to i32
	%241 = icmp ne i32 %240, 0
	br i1 %241, label %296, label %304

242:
	%243 = load i32, i32* %231
	%244 = icmp slt i32 %243, 2
	%245 = zext i1 %244 to i32
	%246 = icmp ne i32 %245, 0
	%247 = zext i1 %246 to i32
	%248 = icmp ne i32 %247, 0
	br i1 %248, label %276, label %283

249:
	%250 = alloca i32
	%251 = load i32, i32* %11
	%252 = load i32*, i32** %1
	%253 = getelementptr i8, i32* %252, i32 %251
	%254 = load i8, i8* %253
	%255 = sext i8 %254 to i32
	%256 = and i32 %255, 255
	%257 = call i32 @rt_hexval(i32 %256)
	store i32 %257, i32* %250
	%258 = load i32, i32* %250
	%259 = icmp slt i32 %258, 0
	%260 = zext i1 %259 to i32
	%261 = icmp ne i32 %260, 0
	br i1 %261, label %286, label %287

262:
	%263 = load i32, i32* %12
	%264 = load i32*, i32** %6
	%265 = getelementptr i8, i32* %264, i32 %263
	%266 = load i32, i32* %230
	%267 = shl i32 %266, 24
	%268 = ashr i32 %267, 24
	%269 = shl i32 %268, 24
	%270 = ashr i32 %269, 24
	%271 = shl i32 %270, 24
	%272 = ashr i32 %271, 24
	%273 = trunc i32 %272 to i8
	store i8 %273, i8* %265
	%274 = load i32, i32* %12
	%275 = add i32 %274, 1
	store i32 %275, i32* %12
	br label %234

276:
	%277 = load i32, i32* %11
	%278 = load i32, i32* %2
	%279 = icmp slt i32 %277, %278
	%280 = zext i1 %279 to i32
	%281 = icmp ne i32 %280, 0
	%282 = zext i1 %281 to i32
	br label %283

283:
	%284 = phi i32 [ %247, %242 ], [ %282, %276 ]
	%285 = icmp ne i32 %284, 0
	br i1 %285, label %249, label %262

286:
	br label %262

287:
	%288 = load i32, i32* %230
	%289 = mul i32 %288, 16
	%290 = load i32, i32* %250
	%291 = add i32 %289, %290
	store i32 %291, i32* %230
	%292 = load i32, i32* %11
	%293 = add i32 %292, 1
	store i32 %293, i32* %11
	%294 = load i32, i32* %231
	%295 = add i32 %294, 1
	store i32 %295, i32* %231
	br label %242

dead75:
	br label %287

296:
	%297 = load i32, i32* %11
	%298 = add i32 %297, 1
	%299 = load i32, i32* %2
	%300 = icmp slt i32 %298, %299
	%301 = zext i1 %300 to i32
	%302 = icmp ne i32 %301, 0
	%303 = zext i1 %302 to i32
	br label %304

304:
	%305 = phi i32 [ %240, %235 ], [ %303, %296 ]
	%306 = icmp ne i32 %305, 0
	br i1 %306, label %307, label %315

307:
	%308 = alloca i32
	%309 = load i32, i32* %27
	store i32 %309, i32* %308
	%310 = load i32, i32* %27
	%311 = icmp eq i32 %310, 97
	%312 = zext i1 %311 to i32
	%313 = icmp ne i32 %312, 0
	br i1 %313, label %331, label %332

314:
	br label %234

315:
	%316 = load i32, i32* %12
	%317 = load i32*, i32** %6
	%318 = getelementptr i8, i32* %317, i32 %316
	%319 = load i32, i32* %20
	%320 = shl i32 %319, 24
	%321 = ashr i32 %320, 24
	%322 = shl i32 %321, 24
	%323 = ashr i32 %322, 24
	%324 = shl i32 %323, 24
	%325 = ashr i32 %324, 24
	%326 = trunc i32 %325 to i8
	store i8 %326, i8* %318
	%327 = load i32, i32* %12
	%328 = add i32 %327, 1
	store i32 %328, i32* %12
	%329 = load i32, i32* %11
	%330 = add i32 %329, 1
	store i32 %330, i32* %11
	br label %314

331:
	store i32 7, i32* %308
	br label %332

332:
	%333 = load i32, i32* %27
	%334 = icmp eq i32 %333, 98
	%335 = zext i1 %334 to i32
	%336 = icmp ne i32 %335, 0
	br i1 %336, label %337, label %338

337:
	store i32 8, i32* %308
	br label %338

338:
	%339 = load i32, i32* %27
	%340 = icmp eq i32 %339, 116
	%341 = zext i1 %340 to i32
	%342 = icmp ne i32 %341, 0
	br i1 %342, label %343, label %344

343:
	store i32 9, i32* %308
	br label %344

344:
	%345 = load i32, i32* %27
	%346 = icmp eq i32 %345, 110
	%347 = zext i1 %346 to i32
	%348 = icmp ne i32 %347, 0
	br i1 %348, label %349, label %350

349:
	store i32 10, i32* %308
	br label %350

350:
	%351 = load i32, i32* %27
	%352 = icmp eq i32 %351, 118
	%353 = zext i1 %352 to i32
	%354 = icmp ne i32 %353, 0
	br i1 %354, label %355, label %356

355:
	store i32 11, i32* %308
	br label %356

356:
	%357 = load i32, i32* %27
	%358 = icmp eq i32 %357, 102
	%359 = zext i1 %358 to i32
	%360 = icmp ne i32 %359, 0
	br i1 %360, label %361, label %362

361:
	store i32 12, i32* %308
	br label %362

362:
	%363 = load i32, i32* %27
	%364 = icmp eq i32 %363, 114
	%365 = zext i1 %364 to i32
	%366 = icmp ne i32 %365, 0
	br i1 %366, label %367, label %368

367:
	store i32 13, i32* %308
	br label %368

368:
	%369 = load i32, i32* %27
	%370 = icmp eq i32 %369, 101
	%371 = zext i1 %370 to i32
	%372 = icmp ne i32 %371, 0
	br i1 %372, label %373, label %374

373:
	store i32 27, i32* %308
	br label %374

374:
	%375 = load i32, i32* %27
	%376 = icmp eq i32 %375, 69
	%377 = zext i1 %376 to i32
	%378 = icmp ne i32 %377, 0
	br i1 %378, label %379, label %380

379:
	store i32 27, i32* %308
	br label %380

380:
	%381 = load i32, i32* %12
	%382 = load i32*, i32** %6
	%383 = getelementptr i8, i32* %382, i32 %381
	%384 = load i32, i32* %308
	%385 = shl i32 %384, 24
	%386 = ashr i32 %385, 24
	%387 = shl i32 %386, 24
	%388 = ashr i32 %387, 24
	%389 = shl i32 %388, 24
	%390 = ashr i32 %389, 24
	%391 = trunc i32 %390 to i8
	store i8 %391, i8* %383
	%392 = load i32, i32* %12
	%393 = add i32 %392, 1
	store i32 %393, i32* %12
	%394 = load i32, i32* %11
	%395 = add i32 %394, 2
	store i32 %395, i32* %11
	br label %314

dead76:
	ret i32* null
}

define i32 @rt_haschar(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = icmp ne i32 1, 0
	br i1 %6, label %7, label %19

7:
	%8 = alloca i32
	%9 = load i32, i32* %4
	%10 = load i32*, i32** %2
	%11 = getelementptr i8, i32* %10, i32 %9
	%12 = load i8, i8* %11
	%13 = sext i8 %12 to i32
	%14 = and i32 %13, 255
	store i32 %14, i32* %8
	%15 = load i32, i32* %8
	%16 = icmp eq i32 %15, 0
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %20, label %21

19:
	ret i32 0

20:
	ret i32 0

21:
	%22 = load i32, i32* %8
	%23 = load i32, i32* %3
	%24 = icmp eq i32 %22, %23
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %27, label %28

dead77:
	br label %21

27:
	ret i32 1

28:
	%29 = load i32, i32* %4
	%30 = add i32 %29, 1
	store i32 %30, i32* %4
	br label %5

dead78:
	br label %28

dead79:
	ret i32 0
}

define i32* @rt_read_line(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32*
	%3 = load i32*, i32** @stdin_buf
	%4 = bitcast i32* %3 to i32*
	store i32* %4, i32** %2
	%5 = alloca i32
	%6 = load i32, i32* @stdin_pos
	store i32 %6, i32* %5
	%7 = alloca i32
	%8 = load i32*, i32** %2
	%9 = bitcast i32* %8 to i32*
	%10 = call i32 @rt_strlen(i32* %9)
	store i32 %10, i32* %7
	store i32 0, i32* @read_eof
	%11 = load i32, i32* %5
	%12 = load i32, i32* %7
	%13 = icmp sge i32 %11, %12
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %16, label %19

16:
	store i32 1, i32* @read_eof
	%17 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%18 = bitcast i8* %17 to i32*
	ret i32* %18

19:
	%20 = alloca i32
	%21 = load i32, i32* %5
	store i32 %21, i32* %20
	br label %22

dead80:
	br label %19

22:
	%23 = icmp ne i32 1, 0
	br i1 %23, label %24, label %50

24:
	%25 = alloca i32
	%26 = load i32, i32* %20
	%27 = load i32, i32* %7
	%28 = icmp sge i32 %26, %27
	%29 = zext i1 %28 to i32
	store i32 %29, i32* %25
	%30 = alloca i8
	%31 = load i32, i32* %20
	%32 = load i32*, i32** %2
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = load i8, i8* %33
	%35 = sext i8 %34 to i32
	%36 = shl i32 %35, 24
	%37 = ashr i32 %36, 24
	%38 = shl i32 %37, 24
	%39 = ashr i32 %38, 24
	%40 = trunc i32 %39 to i8
	store i8 %40, i8* %30
	%41 = alloca i32
	%42 = load i8, i8* %30
	%43 = sext i8 %42 to i32
	%44 = icmp eq i32 %43, 10
	%45 = zext i1 %44 to i32
	store i32 %45, i32* %41
	%46 = load i32, i32* %25
	%47 = icmp ne i32 %46, 0
	%48 = zext i1 %47 to i32
	%49 = icmp ne i32 %48, 0
	br i1 %49, label %69, label %65

50:
	%51 = alloca i32*
	%52 = load i32*, i32** %2
	%53 = bitcast i32* %52 to i32*
	%54 = load i32, i32* %5
	%55 = load i32, i32* %20
	%56 = load i32, i32* %5
	%57 = sub i32 %55, %56
	%58 = call i32* @rt_substr(i32* %53, i32 %54, i32 %57)
	%59 = bitcast i32* %58 to i32*
	store i32* %59, i32** %51
	%60 = load i32, i32* %20
	%61 = load i32, i32* %7
	%62 = icmp slt i32 %60, %61
	%63 = zext i1 %62 to i32
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %76, label %79

65:
	%66 = load i32, i32* %41
	%67 = icmp ne i32 %66, 0
	%68 = zext i1 %67 to i32
	br label %69

69:
	%70 = phi i32 [ %48, %24 ], [ %68, %65 ]
	%71 = icmp ne i32 %70, 0
	br i1 %71, label %72, label %73

72:
	br label %50

73:
	%74 = load i32, i32* %20
	%75 = add i32 %74, 1
	store i32 %75, i32* %20
	br label %22

dead81:
	br label %73

76:
	%77 = load i32, i32* %20
	%78 = add i32 %77, 1
	br label %81

79:
	%80 = load i32, i32* %20
	br label %81

81:
	%82 = phi i32 [ %78, %76 ], [ %80, %79 ]
	store i32 %82, i32* @stdin_pos
	%83 = load i32*, i32** %51
	%84 = bitcast i32* %83 to i32*
	ret i32* %84

dead82:
	ret i32* null
}

define i32* @rt_field(i32* %0, i32* %1, i32 %2, i32 %3) {
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
	%9 = load i32*, i32** %4
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_strlen(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32
	store i32 0, i32* %12
	%13 = alloca i32
	store i32 0, i32* %13
	%14 = alloca i32
	store i32 0, i32* %14
	%15 = alloca i32
	store i32 0, i32* %15
	br label %16

16:
	%17 = load i32, i32* %12
	%18 = load i32, i32* %8
	%19 = icmp slt i32 %17, %18
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	br i1 %21, label %22, label %37

22:
	%23 = alloca i32
	%24 = load i32, i32* %12
	%25 = load i32*, i32** %4
	%26 = getelementptr i8, i32* %25, i32 %24
	%27 = load i8, i8* %26
	%28 = sext i8 %27 to i32
	%29 = and i32 %28, 255
	store i32 %29, i32* %23
	%30 = load i32*, i32** %5
	%31 = bitcast i32* %30 to i32*
	%32 = load i32, i32* %23
	%33 = call i32 @rt_haschar(i32* %31, i32 %32)
	%34 = icmp ne i32 %33, 0
	%35 = zext i1 %34 to i32
	%36 = icmp ne i32 %35, 0
	br i1 %36, label %44, label %52

37:
	%38 = load i32, i32* %15
	%39 = icmp ne i32 %38, 0
	%40 = zext i1 %39 to i32
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	%43 = icmp ne i32 %42, 0
	br i1 %43, label %113, label %120

44:
	%45 = load i32, i32* %15
	%46 = icmp ne i32 %45, 0
	%47 = zext i1 %46 to i32
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %57, label %65

49:
	%50 = load i32, i32* %12
	%51 = add i32 %50, 1
	store i32 %51, i32* %12
	br label %16

52:
	%53 = load i32, i32* %15
	%54 = icmp eq i32 %53, 0
	%55 = zext i1 %54 to i32
	%56 = icmp ne i32 %55, 0
	br i1 %56, label %87, label %96

57:
	%58 = load i32, i32* %13
	%59 = load i32, i32* %6
	%60 = icmp eq i32 %58, %59
	%61 = zext i1 %60 to i32
	%62 = icmp ne i32 %61, 0
	%63 = zext i1 %62 to i32
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %66, label %72

65:
	br label %49

66:
	%67 = load i32, i32* %7
	%68 = icmp eq i32 %67, 0
	%69 = zext i1 %68 to i32
	%70 = icmp ne i32 %69, 0
	%71 = zext i1 %70 to i32
	br label %72

72:
	%73 = phi i32 [ %63, %57 ], [ %71, %66 ]
	%74 = icmp ne i32 %73, 0
	br i1 %74, label %75, label %84

75:
	%76 = load i32*, i32** %4
	%77 = bitcast i32* %76 to i32*
	%78 = load i32, i32* %14
	%79 = load i32, i32* %12
	%80 = load i32, i32* %14
	%81 = sub i32 %79, %80
	%82 = call i32* @rt_substr(i32* %77, i32 %78, i32 %81)
	%83 = bitcast i32* %82 to i32*
	ret i32* %83

84:
	%85 = load i32, i32* %13
	%86 = add i32 %85, 1
	store i32 %86, i32* %13
	store i32 0, i32* %15
	br label %65

dead83:
	br label %84

87:
	store i32 1, i32* %15
	%88 = load i32, i32* %12
	store i32 %88, i32* %14
	%89 = load i32, i32* %13
	%90 = load i32, i32* %6
	%91 = icmp eq i32 %89, %90
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	%94 = zext i1 %93 to i32
	%95 = icmp ne i32 %94, 0
	br i1 %95, label %97, label %103

96:
	br label %49

97:
	%98 = load i32, i32* %7
	%99 = icmp ne i32 %98, 0
	%100 = zext i1 %99 to i32
	%101 = icmp ne i32 %100, 0
	%102 = zext i1 %101 to i32
	br label %103

103:
	%104 = phi i32 [ %94, %87 ], [ %102, %97 ]
	%105 = icmp ne i32 %104, 0
	br i1 %105, label %106, label %112

106:
	%107 = load i32*, i32** %4
	%108 = bitcast i32* %107 to i32*
	%109 = load i32, i32* %12
	%110 = call i32* @rt_substr(i32* %108, i32 %109, i32 -1)
	%111 = bitcast i32* %110 to i32*
	ret i32* %111

112:
	br label %96

dead84:
	br label %112

113:
	%114 = load i32, i32* %13
	%115 = load i32, i32* %6
	%116 = icmp eq i32 %114, %115
	%117 = zext i1 %116 to i32
	%118 = icmp ne i32 %117, 0
	%119 = zext i1 %118 to i32
	br label %120

120:
	%121 = phi i32 [ %42, %37 ], [ %119, %113 ]
	%122 = icmp ne i32 %121, 0
	br i1 %122, label %123, label %129

123:
	%124 = load i32*, i32** %4
	%125 = bitcast i32* %124 to i32*
	%126 = load i32, i32* %14
	%127 = call i32* @rt_substr(i32* %125, i32 %126, i32 -1)
	%128 = bitcast i32* %127 to i32*
	ret i32* %128

129:
	%130 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%131 = bitcast i8* %130 to i32*
	ret i32* %131

dead85:
	br label %129

dead86:
	ret i32* null
}

define i32* @rt_pad(i32* %0, i32 %1, i32 %2, i32 %3) {
entry:
	%4 = alloca i32*
	store i32* %0, i32** %4
	%5 = alloca i32
	store i32 %1, i32* %5
	%6 = alloca i32
	store i32 %2, i32* %6
	%7 = alloca i32
	store i32 %3, i32* %7
	%8 = alloca i32*
	%9 = load i32*, i32** %4
	%10 = bitcast i32* %9 to i32*
	store i32* %10, i32** %8
	br label %11

11:
	%12 = load i32*, i32** %8
	%13 = bitcast i32* %12 to i32*
	%14 = call i32 @rt_strlen(i32* %13)
	%15 = load i32, i32* %5
	%16 = icmp slt i32 %14, %15
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %19, label %27

19:
	%20 = alloca i32*
	%21 = alloca i32*
	%22 = alloca i32*
	%23 = load i32, i32* %7
	%24 = icmp ne i32 %23, 0
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %30, label %54

27:
	%28 = load i32*, i32** %8
	%29 = bitcast i32* %28 to i32*
	ret i32* %29

30:
	%31 = getelementptr [2 x i8], [2 x i8]* @.str.15, i32 0, i32 0
	store i8 48, i8* %31
	%32 = getelementptr [2 x i8], [2 x i8]* @.str.15, i32 0, i32 1
	store i8 0, i8* %32
	%33 = getelementptr [2 x i8], [2 x i8]* @.str.15, i32 0, i32 0
	%34 = bitcast i8* %33 to i32*
	store i32* %34, i32** %20
	br label %35

35:
	%36 = load i32*, i32** %8
	%37 = bitcast i32* %36 to i32*
	%38 = getelementptr [2 x i8], [2 x i8]* @.str.17, i32 0, i32 0
	store i8 32, i8* %38
	%39 = getelementptr [2 x i8], [2 x i8]* @.str.17, i32 0, i32 1
	store i8 0, i8* %39
	%40 = getelementptr [2 x i8], [2 x i8]* @.str.17, i32 0, i32 0
	%41 = bitcast i8* %40 to i32*
	%42 = call i32* @rt_strcat(i32* %37, i32* %41)
	%43 = bitcast i32* %42 to i32*
	store i32* %43, i32** %21
	%44 = load i32*, i32** %20
	%45 = bitcast i32* %44 to i32*
	%46 = load i32*, i32** %8
	%47 = bitcast i32* %46 to i32*
	%48 = call i32* @rt_strcat(i32* %45, i32* %47)
	%49 = bitcast i32* %48 to i32*
	store i32* %49, i32** %22
	%50 = load i32, i32* %6
	%51 = icmp ne i32 %50, 0
	%52 = zext i1 %51 to i32
	%53 = icmp ne i32 %52, 0
	br i1 %53, label %59, label %63

54:
	%55 = getelementptr [2 x i8], [2 x i8]* @.str.16, i32 0, i32 0
	store i8 32, i8* %55
	%56 = getelementptr [2 x i8], [2 x i8]* @.str.16, i32 0, i32 1
	store i8 0, i8* %56
	%57 = getelementptr [2 x i8], [2 x i8]* @.str.16, i32 0, i32 0
	%58 = bitcast i8* %57 to i32*
	store i32* %58, i32** %20
	br label %35

59:
	%60 = load i32*, i32** %21
	%61 = bitcast i32* %60 to i32*
	store i32* %61, i32** %8
	br label %62

62:
	br label %11

63:
	%64 = load i32*, i32** %22
	%65 = bitcast i32* %64 to i32*
	store i32* %65, i32** %8
	br label %62

dead87:
	ret i32* null
}

define i32* @rt_unescape(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32*
	%7 = load i32, i32* %2
	%8 = add i32 %7, 1
	%9 = call i32* @rt_bump(i32 %8)
	%10 = bitcast i32* %9 to i32*
	store i32* %10, i32** %6
	%11 = alloca i32
	store i32 0, i32* %11
	%12 = alloca i32
	store i32 0, i32* %12
	br label %13

13:
	%14 = icmp ne i32 1, 0
	br i1 %14, label %15, label %30

15:
	%16 = alloca i32
	%17 = load i32, i32* %11
	%18 = load i32*, i32** %1
	%19 = getelementptr i8, i32* %18, i32 %17
	%20 = load i8, i8* %19
	%21 = sext i8 %20 to i32
	%22 = and i32 %21, 255
	store i32 %22, i32* %16
	%23 = alloca i32
	%24 = load i32, i32* %16
	%25 = icmp eq i32 %24, 92
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %41, label %49

30:
	%31 = load i32, i32* %12
	%32 = load i32*, i32** %6
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = shl i32 0, 24
	%35 = ashr i32 %34, 24
	%36 = shl i32 %35, 24
	%37 = ashr i32 %36, 24
	%38 = trunc i32 %37 to i8
	store i8 %38, i8* %33
	%39 = load i32*, i32** %6
	%40 = bitcast i32* %39 to i32*
	ret i32* %40

41:
	%42 = load i32, i32* %11
	%43 = add i32 %42, 1
	%44 = load i32, i32* %2
	%45 = icmp slt i32 %43, %44
	%46 = zext i1 %45 to i32
	%47 = icmp ne i32 %46, 0
	%48 = zext i1 %47 to i32
	br label %49

49:
	%50 = phi i32 [ %28, %15 ], [ %48, %41 ]
	store i32 %50, i32* %23
	%51 = alloca i32
	%52 = load i32, i32* %16
	%53 = icmp eq i32 %52, 0
	%54 = zext i1 %53 to i32
	%55 = icmp ne i32 %54, 0
	br i1 %55, label %56, label %57

56:
	br label %30

57:
	%58 = load i32, i32* %23
	%59 = icmp ne i32 %58, 0
	br i1 %59, label %60, label %88

dead88:
	br label %57

60:
	%61 = alloca i32
	%62 = load i32, i32* %11
	%63 = add i32 %62, 1
	%64 = load i32*, i32** %1
	%65 = getelementptr i8, i32* %64, i32 %63
	%66 = load i8, i8* %65
	%67 = sext i8 %66 to i32
	%68 = and i32 %67, 255
	store i32 %68, i32* %61
	%69 = load i32, i32* %61
	store i32 %69, i32* %51
	%70 = load i32, i32* %61
	%71 = icmp eq i32 %70, 110
	%72 = zext i1 %71 to i32
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %92, label %93

74:
	%75 = load i32, i32* %12
	%76 = load i32*, i32** %6
	%77 = getelementptr i8, i32* %76, i32 %75
	%78 = load i32, i32* %51
	%79 = shl i32 %78, 24
	%80 = ashr i32 %79, 24
	%81 = shl i32 %80, 24
	%82 = ashr i32 %81, 24
	%83 = shl i32 %82, 24
	%84 = ashr i32 %83, 24
	%85 = trunc i32 %84 to i8
	store i8 %85, i8* %77
	%86 = load i32, i32* %12
	%87 = add i32 %86, 1
	store i32 %87, i32* %12
	br label %13

88:
	%89 = load i32, i32* %16
	store i32 %89, i32* %51
	%90 = load i32, i32* %11
	%91 = add i32 %90, 1
	store i32 %91, i32* %11
	br label %74

92:
	store i32 10, i32* %51
	br label %93

93:
	%94 = load i32, i32* %61
	%95 = icmp eq i32 %94, 116
	%96 = zext i1 %95 to i32
	%97 = icmp ne i32 %96, 0
	br i1 %97, label %98, label %99

98:
	store i32 9, i32* %51
	br label %99

99:
	%100 = load i32, i32* %61
	%101 = icmp eq i32 %100, 114
	%102 = zext i1 %101 to i32
	%103 = icmp ne i32 %102, 0
	br i1 %103, label %104, label %105

104:
	store i32 13, i32* %51
	br label %105

105:
	%106 = load i32, i32* %61
	%107 = icmp eq i32 %106, 97
	%108 = zext i1 %107 to i32
	%109 = icmp ne i32 %108, 0
	br i1 %109, label %110, label %111

110:
	store i32 7, i32* %51
	br label %111

111:
	%112 = load i32, i32* %61
	%113 = icmp eq i32 %112, 98
	%114 = zext i1 %113 to i32
	%115 = icmp ne i32 %114, 0
	br i1 %115, label %116, label %117

116:
	store i32 8, i32* %51
	br label %117

117:
	%118 = load i32, i32* %61
	%119 = icmp eq i32 %118, 102
	%120 = zext i1 %119 to i32
	%121 = icmp ne i32 %120, 0
	br i1 %121, label %122, label %123

122:
	store i32 12, i32* %51
	br label %123

123:
	%124 = load i32, i32* %61
	%125 = icmp eq i32 %124, 118
	%126 = zext i1 %125 to i32
	%127 = icmp ne i32 %126, 0
	br i1 %127, label %128, label %129

128:
	store i32 11, i32* %51
	br label %129

129:
	%130 = load i32, i32* %61
	%131 = icmp eq i32 %130, 101
	%132 = zext i1 %131 to i32
	%133 = icmp ne i32 %132, 0
	br i1 %133, label %134, label %135

134:
	store i32 27, i32* %51
	br label %135

135:
	%136 = load i32, i32* %61
	%137 = icmp eq i32 %136, 48
	%138 = zext i1 %137 to i32
	%139 = icmp ne i32 %138, 0
	br i1 %139, label %140, label %141

140:
	store i32 0, i32* %51
	br label %141

141:
	%142 = load i32, i32* %11
	%143 = add i32 %142, 2
	store i32 %143, i32* %11
	br label %74

dead89:
	ret i32* null
}

define i32 @rt_nfields(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = getelementptr i8, i32* %3, i32 0
	%5 = load i8, i8* %4
	%6 = sext i8 %5 to i32
	%7 = and i32 %6, 255
	store i32 %7, i32* %2
	%8 = alloca i32
	store i32 1, i32* %8
	%9 = alloca i32
	store i32 0, i32* %9
	%10 = alloca i32
	%11 = load i32*, i32** %1
	%12 = bitcast i32* %11 to i32*
	%13 = call i32 @rt_strlen(i32* %12)
	store i32 %13, i32* %10
	%14 = load i32, i32* %2
	%15 = icmp ne i32 %14, 2
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %18, label %19

18:
	ret i32 1

19:
	br label %20

dead90:
	br label %19

20:
	%21 = load i32, i32* %8
	%22 = load i32, i32* %10
	%23 = icmp slt i32 %21, %22
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %38

26:
	%27 = alloca i32
	%28 = load i32, i32* %8
	%29 = load i32*, i32** %1
	%30 = getelementptr i8, i32* %29, i32 %28
	%31 = load i8, i8* %30
	%32 = sext i8 %31 to i32
	%33 = and i32 %32, 255
	store i32 %33, i32* %27
	%34 = load i32, i32* %27
	%35 = icmp eq i32 %34, 1
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %40, label %43

38:
	%39 = load i32, i32* %9
	ret i32 %39

40:
	%41 = load i32, i32* %9
	%42 = add i32 %41, 1
	store i32 %42, i32* %9
	br label %43

43:
	%44 = load i32, i32* %8
	%45 = add i32 %44, 1
	store i32 %45, i32* %8
	br label %20

dead91:
	ret i32 0
}

define i32* @rt_getfield(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = getelementptr i8, i32* %5, i32 0
	%7 = load i8, i8* %6
	%8 = sext i8 %7 to i32
	%9 = and i32 %8, 255
	store i32 %9, i32* %4
	%10 = alloca i32
	store i32 1, i32* %10
	%11 = alloca i32
	store i32 0, i32* %11
	%12 = alloca i32
	store i32 1, i32* %12
	%13 = alloca i32
	%14 = load i32*, i32** %2
	%15 = bitcast i32* %14 to i32*
	%16 = call i32 @rt_strlen(i32* %15)
	store i32 %16, i32* %13
	%17 = load i32, i32* %4
	%18 = icmp ne i32 %17, 2
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %26

21:
	%22 = load i32, i32* %3
	%23 = icmp eq i32 %22, 0
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %27, label %30

26:
	br label %33

27:
	%28 = load i32*, i32** %2
	%29 = bitcast i32* %28 to i32*
	ret i32* %29

30:
	%31 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%32 = bitcast i8* %31 to i32*
	ret i32* %32

dead92:
	br label %30

dead93:
	br label %26

33:
	%34 = load i32, i32* %10
	%35 = load i32, i32* %13
	%36 = icmp slt i32 %34, %35
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %39, label %51

39:
	%40 = alloca i32
	%41 = load i32, i32* %10
	%42 = load i32*, i32** %2
	%43 = getelementptr i8, i32* %42, i32 %41
	%44 = load i8, i8* %43
	%45 = sext i8 %44 to i32
	%46 = and i32 %45, 255
	store i32 %46, i32* %40
	%47 = load i32, i32* %40
	%48 = icmp eq i32 %47, 1
	%49 = zext i1 %48 to i32
	%50 = icmp ne i32 %49, 0
	br i1 %50, label %54, label %60

51:
	%52 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%53 = bitcast i8* %52 to i32*
	ret i32* %53

54:
	%55 = load i32, i32* %11
	%56 = load i32, i32* %3
	%57 = icmp eq i32 %55, %56
	%58 = zext i1 %57 to i32
	%59 = icmp ne i32 %58, 0
	br i1 %59, label %63, label %72

60:
	%61 = load i32, i32* %10
	%62 = add i32 %61, 1
	store i32 %62, i32* %10
	br label %33

63:
	%64 = load i32*, i32** %2
	%65 = bitcast i32* %64 to i32*
	%66 = load i32, i32* %12
	%67 = load i32, i32* %10
	%68 = load i32, i32* %12
	%69 = sub i32 %67, %68
	%70 = call i32* @rt_substr(i32* %65, i32 %66, i32 %69)
	%71 = bitcast i32* %70 to i32*
	ret i32* %71

72:
	%73 = load i32, i32* %11
	%74 = add i32 %73, 1
	store i32 %74, i32* %11
	%75 = load i32, i32* %10
	%76 = add i32 %75, 1
	store i32 %76, i32* %12
	br label %60

dead94:
	br label %72

dead95:
	ret i32* null
}

define i32* @rt_wordjoin(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = getelementptr i8, i32* %3, i32 0
	%5 = load i8, i8* %4
	%6 = sext i8 %5 to i32
	%7 = and i32 %6, 255
	store i32 %7, i32* %2
	%8 = alloca i32
	%9 = load i32*, i32** %1
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_nfields(i32* %10)
	store i32 %11, i32* %8
	%12 = alloca i32*
	%13 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%14 = bitcast i8* %13 to i32*
	store i32* %14, i32** %12
	%15 = alloca i32
	store i32 0, i32* %15
	%16 = load i32, i32* %2
	%17 = icmp ne i32 %16, 2
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %23

20:
	%21 = load i32*, i32** %1
	%22 = bitcast i32* %21 to i32*
	ret i32* %22

23:
	br label %24

dead96:
	br label %23

24:
	%25 = load i32, i32* %15
	%26 = load i32, i32* %8
	%27 = icmp slt i32 %25, %26
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %30, label %48

30:
	%31 = alloca i32*
	%32 = load i32*, i32** %12
	%33 = bitcast i32* %32 to i32*
	store i32* %33, i32** %31
	%34 = alloca i32*
	%35 = load i32*, i32** %31
	%36 = bitcast i32* %35 to i32*
	%37 = getelementptr [2 x i8], [2 x i8]* @.str.18, i32 0, i32 0
	store i8 32, i8* %37
	%38 = getelementptr [2 x i8], [2 x i8]* @.str.18, i32 0, i32 1
	store i8 0, i8* %38
	%39 = getelementptr [2 x i8], [2 x i8]* @.str.18, i32 0, i32 0
	%40 = bitcast i8* %39 to i32*
	%41 = call i32* @rt_strcat(i32* %36, i32* %40)
	%42 = bitcast i32* %41 to i32*
	store i32* %42, i32** %34
	%43 = alloca i32*
	%44 = load i32, i32* %15
	%45 = icmp sgt i32 %44, 0
	%46 = zext i1 %45 to i32
	%47 = icmp ne i32 %46, 0
	br i1 %47, label %51, label %54

48:
	%49 = load i32*, i32** %12
	%50 = bitcast i32* %49 to i32*
	ret i32* %50

51:
	%52 = load i32*, i32** %34
	%53 = bitcast i32* %52 to i32*
	store i32* %53, i32** %31
	br label %54

54:
	%55 = load i32*, i32** %1
	%56 = bitcast i32* %55 to i32*
	%57 = load i32, i32* %15
	%58 = call i32* @rt_getfield(i32* %56, i32 %57)
	%59 = bitcast i32* %58 to i32*
	store i32* %59, i32** %43
	%60 = load i32*, i32** %31
	%61 = bitcast i32* %60 to i32*
	%62 = load i32*, i32** %43
	%63 = bitcast i32* %62 to i32*
	%64 = call i32* @rt_strcat(i32* %61, i32* %63)
	%65 = bitcast i32* %64 to i32*
	store i32* %65, i32** %12
	%66 = load i32, i32* %15
	%67 = add i32 %66, 1
	store i32 %67, i32* %15
	br label %24

dead97:
	ret i32* null
}

define i32* @rt_splitifs(i32* %0, i32* %1) {
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
	%8 = alloca i32*
	%9 = getelementptr [2 x i8], [2 x i8]* @.str.21, i32 0, i32 0
	store i8 2, i8* %9
	%10 = getelementptr [2 x i8], [2 x i8]* @.str.21, i32 0, i32 1
	store i8 0, i8* %10
	%11 = bitcast [2 x i8]* @.str.21 to i32*
	store i32* %11, i32** %8
	%12 = alloca i32
	store i32 0, i32* %12
	%13 = alloca i32
	store i32 0, i32* %13
	%14 = alloca i32
	store i32 0, i32* %14
	br label %15

15:
	%16 = load i32, i32* %12
	%17 = load i32, i32* %4
	%18 = icmp slt i32 %16, %17
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %36

21:
	%22 = alloca i32
	%23 = load i32, i32* %12
	%24 = load i32*, i32** %2
	%25 = getelementptr i8, i32* %24, i32 %23
	%26 = load i8, i8* %25
	%27 = sext i8 %26 to i32
	%28 = and i32 %27, 255
	store i32 %28, i32* %22
	%29 = load i32*, i32** %3
	%30 = bitcast i32* %29 to i32*
	%31 = load i32, i32* %22
	%32 = call i32 @rt_haschar(i32* %30, i32 %31)
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %41, label %49

36:
	%37 = load i32, i32* %14
	%38 = icmp ne i32 %37, 0
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	br i1 %40, label %77, label %96

41:
	%42 = load i32, i32* %14
	%43 = icmp ne i32 %42, 0
	%44 = zext i1 %43 to i32
	%45 = icmp ne i32 %44, 0
	br i1 %45, label %54, label %73

46:
	%47 = load i32, i32* %12
	%48 = add i32 %47, 1
	store i32 %48, i32* %12
	br label %15

49:
	%50 = load i32, i32* %14
	%51 = icmp eq i32 %50, 0
	%52 = zext i1 %51 to i32
	%53 = icmp ne i32 %52, 0
	br i1 %53, label %74, label %76

54:
	%55 = load i32*, i32** %8
	%56 = bitcast i32* %55 to i32*
	%57 = load i32*, i32** %2
	%58 = bitcast i32* %57 to i32*
	%59 = load i32, i32* %13
	%60 = load i32, i32* %12
	%61 = load i32, i32* %13
	%62 = sub i32 %60, %61
	%63 = call i32* @rt_substr(i32* %58, i32 %59, i32 %62)
	%64 = bitcast i32* %63 to i32*
	%65 = call i32* @rt_strcat(i32* %56, i32* %64)
	%66 = bitcast i32* %65 to i32*
	%67 = getelementptr [2 x i8], [2 x i8]* @.str.19, i32 0, i32 0
	store i8 1, i8* %67
	%68 = getelementptr [2 x i8], [2 x i8]* @.str.19, i32 0, i32 1
	store i8 0, i8* %68
	%69 = getelementptr [2 x i8], [2 x i8]* @.str.19, i32 0, i32 0
	%70 = bitcast i8* %69 to i32*
	%71 = call i32* @rt_strcat(i32* %66, i32* %70)
	%72 = bitcast i32* %71 to i32*
	store i32* %72, i32** %8
	store i32 0, i32* %14
	br label %73

73:
	br label %46

74:
	store i32 1, i32* %14
	%75 = load i32, i32* %12
	store i32 %75, i32* %13
	br label %76

76:
	br label %46

77:
	%78 = load i32*, i32** %8
	%79 = bitcast i32* %78 to i32*
	%80 = load i32*, i32** %2
	%81 = bitcast i32* %80 to i32*
	%82 = load i32, i32* %13
	%83 = load i32, i32* %4
	%84 = load i32, i32* %13
	%85 = sub i32 %83, %84
	%86 = call i32* @rt_substr(i32* %81, i32 %82, i32 %85)
	%87 = bitcast i32* %86 to i32*
	%88 = call i32* @rt_strcat(i32* %79, i32* %87)
	%89 = bitcast i32* %88 to i32*
	%90 = getelementptr [2 x i8], [2 x i8]* @.str.20, i32 0, i32 0
	store i8 1, i8* %90
	%91 = getelementptr [2 x i8], [2 x i8]* @.str.20, i32 0, i32 1
	store i8 0, i8* %91
	%92 = getelementptr [2 x i8], [2 x i8]* @.str.20, i32 0, i32 0
	%93 = bitcast i8* %92 to i32*
	%94 = call i32* @rt_strcat(i32* %89, i32* %93)
	%95 = bitcast i32* %94 to i32*
	store i32* %95, i32** %8
	br label %96

96:
	%97 = load i32*, i32** %8
	%98 = bitcast i32* %97 to i32*
	ret i32* %98

dead98:
	ret i32* null
}

define i32* @rt_bnd_acc(i32* %0, i32* %1, i32* %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32*
	store i32* %1, i32** %4
	%5 = alloca i32*
	store i32* %2, i32** %5
	%6 = alloca i32
	%7 = load i32*, i32** %5
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_nfields(i32* %8)
	store i32 %9, i32* %6
	%10 = alloca i32*
	%11 = load i32*, i32** %3
	%12 = bitcast i32* %11 to i32*
	store i32* %12, i32** %10
	%13 = alloca i32
	store i32 1, i32* %13
	%14 = load i32, i32* %6
	%15 = icmp slt i32 %14, 2
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %18, label %21

18:
	%19 = load i32*, i32** %10
	%20 = bitcast i32* %19 to i32*
	ret i32* %20

21:
	%22 = load i32*, i32** %3
	%23 = bitcast i32* %22 to i32*
	%24 = load i32*, i32** %4
	%25 = bitcast i32* %24 to i32*
	%26 = call i32* @rt_strcat(i32* %23, i32* %25)
	%27 = bitcast i32* %26 to i32*
	%28 = load i32*, i32** %5
	%29 = bitcast i32* %28 to i32*
	%30 = call i32* @rt_getfield(i32* %29, i32 0)
	%31 = bitcast i32* %30 to i32*
	%32 = call i32* @rt_strcat(i32* %27, i32* %31)
	%33 = bitcast i32* %32 to i32*
	%34 = getelementptr [2 x i8], [2 x i8]* @.str.22, i32 0, i32 0
	store i8 1, i8* %34
	%35 = getelementptr [2 x i8], [2 x i8]* @.str.22, i32 0, i32 1
	store i8 0, i8* %35
	%36 = getelementptr [2 x i8], [2 x i8]* @.str.22, i32 0, i32 0
	%37 = bitcast i8* %36 to i32*
	%38 = call i32* @rt_strcat(i32* %33, i32* %37)
	%39 = bitcast i32* %38 to i32*
	store i32* %39, i32** %10
	br label %40

dead99:
	br label %21

40:
	%41 = load i32, i32* %13
	%42 = load i32, i32* %6
	%43 = sub i32 %42, 1
	%44 = icmp slt i32 %41, %43
	%45 = zext i1 %44 to i32
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %65

47:
	%48 = load i32*, i32** %10
	%49 = bitcast i32* %48 to i32*
	%50 = load i32*, i32** %5
	%51 = bitcast i32* %50 to i32*
	%52 = load i32, i32* %13
	%53 = call i32* @rt_getfield(i32* %51, i32 %52)
	%54 = bitcast i32* %53 to i32*
	%55 = call i32* @rt_strcat(i32* %49, i32* %54)
	%56 = bitcast i32* %55 to i32*
	%57 = getelementptr [2 x i8], [2 x i8]* @.str.23, i32 0, i32 0
	store i8 1, i8* %57
	%58 = getelementptr [2 x i8], [2 x i8]* @.str.23, i32 0, i32 1
	store i8 0, i8* %58
	%59 = getelementptr [2 x i8], [2 x i8]* @.str.23, i32 0, i32 0
	%60 = bitcast i8* %59 to i32*
	%61 = call i32* @rt_strcat(i32* %56, i32* %60)
	%62 = bitcast i32* %61 to i32*
	store i32* %62, i32** %10
	%63 = load i32, i32* %13
	%64 = add i32 %63, 1
	store i32 %64, i32* %13
	br label %40

65:
	%66 = load i32*, i32** %10
	%67 = bitcast i32* %66 to i32*
	ret i32* %67

dead100:
	ret i32* null
}

define i32* @rt_bnd_open(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %3
	%6 = bitcast i32* %5 to i32*
	%7 = call i32 @rt_nfields(i32* %6)
	store i32 %7, i32* %4
	%8 = load i32, i32* %4
	%9 = icmp eq i32 %8, 0
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %15

12:
	%13 = load i32*, i32** %2
	%14 = bitcast i32* %13 to i32*
	ret i32* %14

15:
	%16 = load i32, i32* %4
	%17 = icmp eq i32 %16, 1
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %29

dead101:
	br label %15

20:
	%21 = load i32*, i32** %2
	%22 = bitcast i32* %21 to i32*
	%23 = load i32*, i32** %3
	%24 = bitcast i32* %23 to i32*
	%25 = call i32* @rt_getfield(i32* %24, i32 0)
	%26 = bitcast i32* %25 to i32*
	%27 = call i32* @rt_strcat(i32* %22, i32* %26)
	%28 = bitcast i32* %27 to i32*
	ret i32* %28

29:
	%30 = load i32*, i32** %3
	%31 = bitcast i32* %30 to i32*
	%32 = load i32, i32* %4
	%33 = sub i32 %32, 1
	%34 = call i32* @rt_getfield(i32* %31, i32 %33)
	%35 = bitcast i32* %34 to i32*
	ret i32* %35

dead102:
	br label %29

dead103:
	ret i32* null
}

define i32 @rt_arr_find(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32, i32* @arr_n
	store i32 %5, i32* %4
	%6 = alloca i32
	store i32 0, i32* %6
	br label %7

7:
	%8 = load i32, i32* %6
	%9 = load i32, i32* %4
	%10 = icmp slt i32 %8, %9
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %13, label %38

13:
	%14 = alloca i32
	%15 = load i32*, i32** %2
	%16 = bitcast i32* %15 to i32*
	%17 = load i32, i32* %6
	%18 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %17
	%19 = load i32*, i32** %18
	%20 = bitcast i32* %19 to i32*
	%21 = call i32 @rt_streq(i32* %16, i32* %20)
	%22 = icmp ne i32 %21, 0
	%23 = zext i1 %22 to i32
	store i32 %23, i32* %14
	%24 = alloca i32
	%25 = load i32*, i32** %3
	%26 = bitcast i32* %25 to i32*
	%27 = load i32, i32* %6
	%28 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %27
	%29 = load i32*, i32** %28
	%30 = bitcast i32* %29 to i32*
	%31 = call i32 @rt_streq(i32* %26, i32* %30)
	%32 = icmp ne i32 %31, 0
	%33 = zext i1 %32 to i32
	store i32 %33, i32* %24
	%34 = load i32, i32* %14
	%35 = icmp ne i32 %34, 0
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %39, label %43

38:
	ret i32 -1

39:
	%40 = load i32, i32* %24
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	br label %43

43:
	%44 = phi i32 [ %36, %13 ], [ %42, %39 ]
	%45 = icmp ne i32 %44, 0
	br i1 %45, label %46, label %48

46:
	%47 = load i32, i32* %6
	ret i32 %47

48:
	%49 = load i32, i32* %6
	%50 = add i32 %49, 1
	store i32 %50, i32* %6
	br label %7

dead104:
	br label %48

dead105:
	ret i32 0
}

define i32 @rt_arr_set(i32* %0, i32* %1, i32* %2) {
entry:
	%3 = alloca i32*
	store i32* %0, i32** %3
	%4 = alloca i32*
	store i32* %1, i32** %4
	%5 = alloca i32*
	store i32* %2, i32** %5
	%6 = alloca i32
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = load i32*, i32** %4
	%10 = bitcast i32* %9 to i32*
	%11 = call i32 @rt_arr_find(i32* %8, i32* %10)
	store i32 %11, i32* %6
	%12 = load i32, i32* %6
	%13 = icmp sge i32 %12, 0
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %16, label %21

16:
	%17 = load i32, i32* %6
	%18 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %17
	%19 = load i32*, i32** %5
	%20 = bitcast i32* %19 to i32*
	store i32* %20, i32** %18
	ret i32 0

21:
	%22 = alloca i32
	%23 = load i32, i32* @arr_n
	store i32 %23, i32* %22
	%24 = load i32, i32* %22
	%25 = icmp sge i32 %24, u0x1000
	%26 = zext i1 %25 to i32
	%27 = icmp ne i32 %26, 0
	br i1 %27, label %28, label %29

dead106:
	br label %21

28:
	store i32 1, i32* @rt_limit
	ret i32 0

29:
	%30 = load i32, i32* %22
	%31 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %30
	%32 = load i32*, i32** %3
	%33 = bitcast i32* %32 to i32*
	store i32* %33, i32** %31
	%34 = load i32, i32* %22
	%35 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %34
	%36 = load i32*, i32** %4
	%37 = bitcast i32* %36 to i32*
	store i32* %37, i32** %35
	%38 = load i32, i32* %22
	%39 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %38
	%40 = load i32*, i32** %5
	%41 = bitcast i32* %40 to i32*
	store i32* %41, i32** %39
	%42 = load i32, i32* %22
	%43 = add i32 %42, 1
	store i32 %43, i32* @arr_n
	ret i32 0

dead107:
	br label %29

dead108:
	ret i32 0
}

define i32* @rt_arr_get(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_arr_find(i32* %6, i32* %8)
	store i32 %9, i32* %4
	%10 = load i32, i32* %4
	%11 = icmp sge i32 %10, 0
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	br i1 %13, label %14, label %19

14:
	%15 = load i32, i32* %4
	%16 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %15
	%17 = load i32*, i32** %16
	%18 = bitcast i32* %17 to i32*
	ret i32* %18

19:
	%20 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%21 = bitcast i8* %20 to i32*
	ret i32* %21

dead109:
	br label %19

dead110:
	ret i32* null
}

define i32 @rt_arr_has(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_arr_find(i32* %6, i32* %8)
	store i32 %9, i32* %4
	%10 = load i32, i32* %4
	%11 = icmp sge i32 %10, 0
	%12 = zext i1 %11 to i32
	ret i32 %12

dead111:
	ret i32 0
}

define i32 @rt_arr_del(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %2
	%6 = bitcast i32* %5 to i32*
	%7 = load i32*, i32** %3
	%8 = bitcast i32* %7 to i32*
	%9 = call i32 @rt_arr_find(i32* %6, i32* %8)
	store i32 %9, i32* %4
	%10 = alloca i32
	%11 = load i32, i32* %4
	store i32 %11, i32* %10
	%12 = load i32, i32* %4
	%13 = icmp sge i32 %12, 0
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %16, label %17

16:
	br label %18

17:
	ret i32 0

18:
	%19 = icmp ne i32 1, 0
	br i1 %19, label %20, label %29

20:
	%21 = alloca i32
	%22 = load i32, i32* @arr_n
	store i32 %22, i32* %21
	%23 = load i32, i32* %10
	%24 = load i32, i32* %21
	%25 = sub i32 %24, 1
	%26 = icmp slt i32 %23, %25
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %30, label %54

29:
	br label %17

30:
	%31 = alloca i32
	%32 = load i32, i32* %10
	%33 = add i32 %32, 1
	store i32 %33, i32* %31
	%34 = load i32, i32* %10
	%35 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %34
	%36 = load i32, i32* %31
	%37 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %36
	%38 = load i32*, i32** %37
	%39 = bitcast i32* %38 to i32*
	store i32* %39, i32** %35
	%40 = load i32, i32* %10
	%41 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %40
	%42 = load i32, i32* %31
	%43 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %42
	%44 = load i32*, i32** %43
	%45 = bitcast i32* %44 to i32*
	store i32* %45, i32** %41
	%46 = load i32, i32* %10
	%47 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %46
	%48 = load i32, i32* %31
	%49 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %48
	%50 = load i32*, i32** %49
	%51 = bitcast i32* %50 to i32*
	store i32* %51, i32** %47
	%52 = load i32, i32* %31
	store i32 %52, i32* %10
	br label %53

53:
	br label %18

54:
	%55 = load i32, i32* @arr_n
	%56 = sub i32 %55, 1
	store i32 %56, i32* @arr_n
	br label %29

dead112:
	br label %53

dead113:
	ret i32 0
}

define i32 @rt_arr_count(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32, i32* @arr_n
	store i32 %3, i32* %2
	%4 = alloca i32
	store i32 0, i32* %4
	%5 = alloca i32
	store i32 0, i32* %5
	br label %6

6:
	%7 = load i32, i32* %4
	%8 = load i32, i32* %2
	%9 = icmp slt i32 %7, %8
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %25

12:
	%13 = alloca i32
	%14 = load i32*, i32** %1
	%15 = bitcast i32* %14 to i32*
	%16 = load i32, i32* %4
	%17 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %16
	%18 = load i32*, i32** %17
	%19 = bitcast i32* %18 to i32*
	%20 = call i32 @rt_streq(i32* %15, i32* %19)
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	store i32 %22, i32* %13
	%23 = load i32, i32* %13
	%24 = icmp ne i32 %23, 0
	br i1 %24, label %27, label %30

25:
	%26 = load i32, i32* %5
	ret i32 %26

27:
	%28 = load i32, i32* %5
	%29 = add i32 %28, 1
	br label %32

30:
	%31 = load i32, i32* %5
	br label %32

32:
	%33 = phi i32 [ %29, %27 ], [ %31, %30 ]
	store i32 %33, i32* %5
	%34 = load i32, i32* %4
	%35 = add i32 %34, 1
	store i32 %35, i32* %4
	br label %6

dead114:
	ret i32 0
}

define i32* @rt_arr_list(i32* %0, i32* %1, i32 %2, i32 %3) {
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
	%9 = load i32, i32* @arr_n
	store i32 %9, i32* %8
	%10 = alloca i32
	%11 = load i32, i32* %6
	%12 = icmp ne i32 %11, 0
	%13 = zext i1 %12 to i32
	store i32 %13, i32* %10
	%14 = alloca i32*
	%15 = load i32, i32* %10
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %17, label %20

17:
	%18 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%19 = bitcast i8* %18 to i32*
	br label %25

20:
	%21 = getelementptr [2 x i8], [2 x i8]* @.str.24, i32 0, i32 0
	store i8 2, i8* %21
	%22 = getelementptr [2 x i8], [2 x i8]* @.str.24, i32 0, i32 1
	store i8 0, i8* %22
	%23 = getelementptr [2 x i8], [2 x i8]* @.str.24, i32 0, i32 0
	%24 = bitcast i8* %23 to i32*
	br label %25

25:
	%26 = phi i32* [ %19, %17 ], [ %24, %20 ]
	%27 = bitcast i32* %26 to i32*
	store i32* %27, i32** %14
	%28 = alloca i32
	store i32 0, i32* %28
	%29 = alloca i32
	store i32 0, i32* %29
	br label %30

30:
	%31 = load i32, i32* %28
	%32 = load i32, i32* %8
	%33 = icmp slt i32 %31, %32
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %36, label %49

36:
	%37 = alloca i32
	%38 = load i32*, i32** %4
	%39 = bitcast i32* %38 to i32*
	%40 = load i32, i32* %28
	%41 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %40
	%42 = load i32*, i32** %41
	%43 = bitcast i32* %42 to i32*
	%44 = call i32 @rt_streq(i32* %39, i32* %43)
	%45 = icmp ne i32 %44, 0
	%46 = zext i1 %45 to i32
	store i32 %46, i32* %37
	%47 = load i32, i32* %37
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %52, label %58

49:
	%50 = load i32*, i32** %14
	%51 = bitcast i32* %50 to i32*
	ret i32* %51

52:
	%53 = alloca i32*
	%54 = load i32, i32* %7
	%55 = icmp ne i32 %54, 0
	%56 = zext i1 %55 to i32
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %61, label %66

58:
	%59 = load i32, i32* %28
	%60 = add i32 %59, 1
	store i32 %60, i32* %28
	br label %30

61:
	%62 = load i32, i32* %28
	%63 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %62
	%64 = load i32*, i32** %63
	%65 = bitcast i32* %64 to i32*
	br label %71

66:
	%67 = load i32, i32* %28
	%68 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %67
	%69 = load i32*, i32** %68
	%70 = bitcast i32* %69 to i32*
	br label %71

71:
	%72 = phi i32* [ %65, %61 ], [ %70, %66 ]
	%73 = bitcast i32* %72 to i32*
	store i32* %73, i32** %53
	%74 = alloca i32*
	%75 = load i32*, i32** %14
	%76 = bitcast i32* %75 to i32*
	store i32* %76, i32** %74
	%77 = alloca i32
	%78 = load i32, i32* %10
	%79 = icmp ne i32 %78, 0
	%80 = zext i1 %79 to i32
	%81 = icmp ne i32 %80, 0
	br i1 %81, label %82, label %88

82:
	%83 = load i32, i32* %29
	%84 = icmp sgt i32 %83, 0
	%85 = zext i1 %84 to i32
	%86 = icmp ne i32 %85, 0
	%87 = zext i1 %86 to i32
	br label %88

88:
	%89 = phi i32 [ %80, %71 ], [ %87, %82 ]
	store i32 %89, i32* %77
	%90 = alloca i32*
	%91 = load i32*, i32** %74
	%92 = bitcast i32* %91 to i32*
	%93 = load i32*, i32** %5
	%94 = bitcast i32* %93 to i32*
	%95 = call i32* @rt_strcat(i32* %92, i32* %94)
	%96 = bitcast i32* %95 to i32*
	store i32* %96, i32** %90
	%97 = load i32, i32* %77
	%98 = icmp ne i32 %97, 0
	br i1 %98, label %99, label %102

99:
	%100 = load i32*, i32** %90
	%101 = bitcast i32* %100 to i32*
	store i32* %101, i32** %74
	br label %102

102:
	%103 = load i32*, i32** %74
	%104 = bitcast i32* %103 to i32*
	%105 = load i32*, i32** %53
	%106 = bitcast i32* %105 to i32*
	%107 = call i32* @rt_strcat(i32* %104, i32* %106)
	%108 = bitcast i32* %107 to i32*
	store i32* %108, i32** %74
	%109 = alloca i32*
	%110 = load i32*, i32** %74
	%111 = bitcast i32* %110 to i32*
	%112 = getelementptr [2 x i8], [2 x i8]* @.str.25, i32 0, i32 0
	store i8 1, i8* %112
	%113 = getelementptr [2 x i8], [2 x i8]* @.str.25, i32 0, i32 1
	store i8 0, i8* %113
	%114 = getelementptr [2 x i8], [2 x i8]* @.str.25, i32 0, i32 0
	%115 = bitcast i8* %114 to i32*
	%116 = call i32* @rt_strcat(i32* %111, i32* %115)
	%117 = bitcast i32* %116 to i32*
	store i32* %117, i32** %109
	%118 = load i32, i32* %10
	%119 = icmp eq i32 %118, 0
	%120 = zext i1 %119 to i32
	%121 = icmp ne i32 %120, 0
	br i1 %121, label %122, label %125

122:
	%123 = load i32*, i32** %109
	%124 = bitcast i32* %123 to i32*
	store i32* %124, i32** %74
	br label %125

125:
	%126 = load i32*, i32** %74
	%127 = bitcast i32* %126 to i32*
	store i32* %127, i32** %14
	%128 = load i32, i32* %29
	%129 = add i32 %128, 1
	store i32 %129, i32* %29
	br label %58

dead115:
	ret i32* null
}

define i32 @rt_arr_nextidx(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32, i32* @arr_n
	store i32 %3, i32* %2
	%4 = alloca i32
	store i32 0, i32* %4
	%5 = alloca i32
	store i32 0, i32* %5
	br label %6

6:
	%7 = load i32, i32* %4
	%8 = load i32, i32* %2
	%9 = icmp slt i32 %7, %8
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %34

12:
	%13 = alloca i32
	%14 = load i32*, i32** %1
	%15 = bitcast i32* %14 to i32*
	%16 = load i32, i32* %4
	%17 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %16
	%18 = load i32*, i32** %17
	%19 = bitcast i32* %18 to i32*
	%20 = call i32 @rt_streq(i32* %15, i32* %19)
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	store i32 %22, i32* %13
	%23 = alloca i32
	%24 = load i32, i32* %4
	%25 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %24
	%26 = load i32*, i32** %25
	%27 = bitcast i32* %26 to i32*
	%28 = call i32 @rt_str2int(i32* %27)
	store i32 %28, i32* %23
	%29 = alloca i32
	%30 = load i32, i32* %13
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	br i1 %33, label %36, label %43

34:
	%35 = load i32, i32* %5
	ret i32 %35

36:
	%37 = load i32, i32* %23
	%38 = load i32, i32* %5
	%39 = icmp sge i32 %37, %38
	%40 = zext i1 %39 to i32
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	br label %43

43:
	%44 = phi i32 [ %32, %12 ], [ %42, %36 ]
	store i32 %44, i32* %29
	%45 = load i32, i32* %29
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %50

47:
	%48 = load i32, i32* %23
	%49 = add i32 %48, 1
	br label %52

50:
	%51 = load i32, i32* %5
	br label %52

52:
	%53 = phi i32 [ %49, %47 ], [ %51, %50 ]
	store i32 %53, i32* %5
	%54 = load i32, i32* %4
	%55 = add i32 %54, 1
	store i32 %55, i32* %4
	br label %6

dead116:
	ret i32 0
}

define i32 @rt_arr_clear(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = alloca i32
	store i32 0, i32* %3
	br label %4

4:
	%5 = load i32, i32* %2
	%6 = load i32, i32* @arr_n
	%7 = icmp slt i32 %5, %6
	%8 = zext i1 %7 to i32
	%9 = icmp ne i32 %8, 0
	br i1 %9, label %10, label %25

10:
	%11 = alloca i32
	%12 = load i32*, i32** %1
	%13 = bitcast i32* %12 to i32*
	%14 = load i32, i32* %2
	%15 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %14
	%16 = load i32*, i32** %15
	%17 = bitcast i32* %16 to i32*
	%18 = call i32 @rt_streq(i32* %13, i32* %17)
	%19 = icmp ne i32 %18, 0
	%20 = zext i1 %19 to i32
	store i32 %20, i32* %11
	%21 = load i32, i32* %11
	%22 = icmp eq i32 %21, 0
	%23 = zext i1 %22 to i32
	%24 = icmp ne i32 %23, 0
	br i1 %24, label %27, label %48

25:
	%26 = load i32, i32* %3
	store i32 %26, i32* @arr_n
	ret i32 0

27:
	%28 = load i32, i32* %3
	%29 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %28
	%30 = load i32, i32* %2
	%31 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_nm, i32 0, i32 %30
	%32 = load i32*, i32** %31
	%33 = bitcast i32* %32 to i32*
	store i32* %33, i32** %29
	%34 = load i32, i32* %3
	%35 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %34
	%36 = load i32, i32* %2
	%37 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_k, i32 0, i32 %36
	%38 = load i32*, i32** %37
	%39 = bitcast i32* %38 to i32*
	store i32* %39, i32** %35
	%40 = load i32, i32* %3
	%41 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %40
	%42 = load i32, i32* %2
	%43 = getelementptr [4096 x i32*], [4096 x i32*]* @arr_v, i32 0, i32 %42
	%44 = load i32*, i32** %43
	%45 = bitcast i32* %44 to i32*
	store i32* %45, i32** %41
	%46 = load i32, i32* %3
	%47 = add i32 %46, 1
	store i32 %47, i32* %3
	br label %48

48:
	%49 = load i32, i32* %2
	%50 = add i32 %49, 1
	store i32 %50, i32* %2
	br label %4

dead117:
	ret i32 0
}

define i32 @rt_arr_append(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %3
	%6 = bitcast i32* %5 to i32*
	%7 = call i32 @rt_nfields(i32* %6)
	store i32 %7, i32* %4
	%8 = alloca i32
	store i32 0, i32* %8
	br label %9

9:
	%10 = load i32, i32* %8
	%11 = load i32, i32* %4
	%12 = icmp slt i32 %10, %11
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	br i1 %14, label %15, label %39

15:
	%16 = alloca i32
	%17 = load i32*, i32** %2
	%18 = bitcast i32* %17 to i32*
	%19 = call i32 @rt_arr_nextidx(i32* %18)
	store i32 %19, i32* %16
	%20 = alloca i32*
	%21 = load i32, i32* %16
	%22 = call i32* @rt_int2str(i32 %21)
	%23 = bitcast i32* %22 to i32*
	store i32* %23, i32** %20
	%24 = alloca i32*
	%25 = load i32*, i32** %3
	%26 = bitcast i32* %25 to i32*
	%27 = load i32, i32* %8
	%28 = call i32* @rt_getfield(i32* %26, i32 %27)
	%29 = bitcast i32* %28 to i32*
	store i32* %29, i32** %24
	%30 = load i32*, i32** %2
	%31 = bitcast i32* %30 to i32*
	%32 = load i32*, i32** %20
	%33 = bitcast i32* %32 to i32*
	%34 = load i32*, i32** %24
	%35 = bitcast i32* %34 to i32*
	%36 = call i32 @rt_arr_set(i32* %31, i32* %33, i32* %35)
	%37 = load i32, i32* %8
	%38 = add i32 %37, 1
	store i32 %38, i32* %8
	br label %9

39:
	ret i32 0

dead118:
	ret i32 0
}

define i32* @rt_slicefields(i32* %0, i32 %1, i32 %2, i32* %3, i32 %4) {
entry:
	%5 = alloca i32*
	store i32* %0, i32** %5
	%6 = alloca i32
	store i32 %1, i32* %6
	%7 = alloca i32
	store i32 %2, i32* %7
	%8 = alloca i32*
	store i32* %3, i32** %8
	%9 = alloca i32
	store i32 %4, i32* %9
	%10 = alloca i32
	%11 = load i32*, i32** %5
	%12 = bitcast i32* %11 to i32*
	%13 = call i32 @rt_nfields(i32* %12)
	store i32 %13, i32* %10
	%14 = alloca i32
	%15 = load i32, i32* %9
	%16 = icmp ne i32 %15, 0
	%17 = zext i1 %16 to i32
	store i32 %17, i32* %14
	%18 = alloca i32*
	%19 = load i32, i32* %14
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %24

21:
	%22 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%23 = bitcast i8* %22 to i32*
	br label %29

24:
	%25 = getelementptr [2 x i8], [2 x i8]* @.str.26, i32 0, i32 0
	store i8 2, i8* %25
	%26 = getelementptr [2 x i8], [2 x i8]* @.str.26, i32 0, i32 1
	store i8 0, i8* %26
	%27 = getelementptr [2 x i8], [2 x i8]* @.str.26, i32 0, i32 0
	%28 = bitcast i8* %27 to i32*
	br label %29

29:
	%30 = phi i32* [ %23, %21 ], [ %28, %24 ]
	%31 = bitcast i32* %30 to i32*
	store i32* %31, i32** %18
	%32 = alloca i32
	%33 = load i32, i32* %6
	store i32 %33, i32* %32
	%34 = alloca i32
	store i32 0, i32* %34
	br label %35

35:
	%36 = load i32, i32* %32
	%37 = load i32, i32* %10
	%38 = icmp slt i32 %36, %37
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	%41 = zext i1 %40 to i32
	%42 = icmp ne i32 %41, 0
	br i1 %42, label %55, label %62

43:
	%44 = alloca i32*
	%45 = load i32*, i32** %18
	%46 = bitcast i32* %45 to i32*
	store i32* %46, i32** %44
	%47 = alloca i32
	%48 = load i32, i32* %14
	%49 = icmp ne i32 %48, 0
	%50 = zext i1 %49 to i32
	%51 = icmp ne i32 %50, 0
	br i1 %51, label %65, label %71

52:
	%53 = load i32*, i32** %18
	%54 = bitcast i32* %53 to i32*
	ret i32* %54

55:
	%56 = load i32, i32* %34
	%57 = load i32, i32* %7
	%58 = icmp slt i32 %56, %57
	%59 = zext i1 %58 to i32
	%60 = icmp ne i32 %59, 0
	%61 = zext i1 %60 to i32
	br label %62

62:
	%63 = phi i32 [ %41, %35 ], [ %61, %55 ]
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %43, label %52

65:
	%66 = load i32, i32* %34
	%67 = icmp sgt i32 %66, 0
	%68 = zext i1 %67 to i32
	%69 = icmp ne i32 %68, 0
	%70 = zext i1 %69 to i32
	br label %71

71:
	%72 = phi i32 [ %50, %43 ], [ %70, %65 ]
	store i32 %72, i32* %47
	%73 = alloca i32*
	%74 = load i32*, i32** %44
	%75 = bitcast i32* %74 to i32*
	%76 = load i32*, i32** %8
	%77 = bitcast i32* %76 to i32*
	%78 = call i32* @rt_strcat(i32* %75, i32* %77)
	%79 = bitcast i32* %78 to i32*
	store i32* %79, i32** %73
	%80 = load i32, i32* %47
	%81 = icmp ne i32 %80, 0
	br i1 %81, label %82, label %85

82:
	%83 = load i32*, i32** %73
	%84 = bitcast i32* %83 to i32*
	store i32* %84, i32** %44
	br label %85

85:
	%86 = alloca i32*
	%87 = load i32*, i32** %5
	%88 = bitcast i32* %87 to i32*
	%89 = load i32, i32* %32
	%90 = call i32* @rt_getfield(i32* %88, i32 %89)
	%91 = bitcast i32* %90 to i32*
	store i32* %91, i32** %86
	%92 = load i32*, i32** %44
	%93 = bitcast i32* %92 to i32*
	%94 = load i32*, i32** %86
	%95 = bitcast i32* %94 to i32*
	%96 = call i32* @rt_strcat(i32* %93, i32* %95)
	%97 = bitcast i32* %96 to i32*
	store i32* %97, i32** %44
	%98 = alloca i32*
	%99 = load i32*, i32** %44
	%100 = bitcast i32* %99 to i32*
	%101 = getelementptr [2 x i8], [2 x i8]* @.str.27, i32 0, i32 0
	store i8 1, i8* %101
	%102 = getelementptr [2 x i8], [2 x i8]* @.str.27, i32 0, i32 1
	store i8 0, i8* %102
	%103 = getelementptr [2 x i8], [2 x i8]* @.str.27, i32 0, i32 0
	%104 = bitcast i8* %103 to i32*
	%105 = call i32* @rt_strcat(i32* %100, i32* %104)
	%106 = bitcast i32* %105 to i32*
	store i32* %106, i32** %98
	%107 = load i32, i32* %14
	%108 = icmp eq i32 %107, 0
	%109 = zext i1 %108 to i32
	%110 = icmp ne i32 %109, 0
	br i1 %110, label %111, label %114

111:
	%112 = load i32*, i32** %98
	%113 = bitcast i32* %112 to i32*
	store i32* %113, i32** %44
	br label %114

114:
	%115 = load i32*, i32** %44
	%116 = bitcast i32* %115 to i32*
	store i32* %116, i32** %18
	%117 = load i32, i32* %32
	%118 = add i32 %117, 1
	store i32 %118, i32* %32
	%119 = load i32, i32* %34
	%120 = add i32 %119, 1
	store i32 %120, i32* %34
	br label %35

dead119:
	ret i32* null
}

define i32* @rt_globescape(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_strlen(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32*
	%7 = load i32, i32* %2
	%8 = mul i32 %7, 2
	%9 = add i32 %8, 1
	%10 = call i32* @rt_bump(i32 %9)
	%11 = bitcast i32* %10 to i32*
	store i32* %11, i32** %6
	%12 = alloca i32
	store i32 0, i32* %12
	%13 = alloca i32
	store i32 0, i32* %13
	br label %14

14:
	%15 = load i32, i32* %12
	%16 = load i32, i32* %2
	%17 = icmp slt i32 %15, %16
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %35

20:
	%21 = alloca i32
	%22 = load i32, i32* %12
	%23 = load i32*, i32** %1
	%24 = getelementptr i8, i32* %23, i32 %22
	%25 = load i8, i8* %24
	%26 = sext i8 %25 to i32
	%27 = and i32 %26, 255
	store i32 %27, i32* %21
	%28 = alloca i32
	%29 = load i32, i32* %21
	%30 = icmp eq i32 %29, 42
	%31 = zext i1 %30 to i32
	%32 = icmp ne i32 %31, 0
	%33 = zext i1 %32 to i32
	%34 = icmp ne i32 %33, 0
	br i1 %34, label %52, label %46

35:
	%36 = load i32, i32* %13
	%37 = load i32*, i32** %6
	%38 = getelementptr i8, i32* %37, i32 %36
	%39 = shl i32 0, 24
	%40 = ashr i32 %39, 24
	%41 = shl i32 %40, 24
	%42 = ashr i32 %41, 24
	%43 = trunc i32 %42 to i8
	store i8 %43, i8* %38
	%44 = load i32*, i32** %6
	%45 = bitcast i32* %44 to i32*
	ret i32* %45

46:
	%47 = load i32, i32* %21
	%48 = icmp eq i32 %47, 63
	%49 = zext i1 %48 to i32
	%50 = icmp ne i32 %49, 0
	%51 = zext i1 %50 to i32
	br label %52

52:
	%53 = phi i32 [ %33, %20 ], [ %51, %46 ]
	%54 = icmp ne i32 %53, 0
	br i1 %54, label %61, label %55

55:
	%56 = load i32, i32* %21
	%57 = icmp eq i32 %56, 91
	%58 = zext i1 %57 to i32
	%59 = icmp ne i32 %58, 0
	%60 = zext i1 %59 to i32
	br label %61

61:
	%62 = phi i32 [ %53, %52 ], [ %60, %55 ]
	%63 = icmp ne i32 %62, 0
	br i1 %63, label %70, label %64

64:
	%65 = load i32, i32* %21
	%66 = icmp eq i32 %65, 93
	%67 = zext i1 %66 to i32
	%68 = icmp ne i32 %67, 0
	%69 = zext i1 %68 to i32
	br label %70

70:
	%71 = phi i32 [ %62, %61 ], [ %69, %64 ]
	%72 = icmp ne i32 %71, 0
	br i1 %72, label %79, label %73

73:
	%74 = load i32, i32* %21
	%75 = icmp eq i32 %74, 92
	%76 = zext i1 %75 to i32
	%77 = icmp ne i32 %76, 0
	%78 = zext i1 %77 to i32
	br label %79

79:
	%80 = phi i32 [ %71, %70 ], [ %78, %73 ]
	store i32 %80, i32* %28
	%81 = load i32, i32* %28
	%82 = icmp ne i32 %81, 0
	br i1 %82, label %83, label %94

83:
	%84 = load i32, i32* %13
	%85 = load i32*, i32** %6
	%86 = getelementptr i8, i32* %85, i32 %84
	%87 = shl i32 92, 24
	%88 = ashr i32 %87, 24
	%89 = shl i32 %88, 24
	%90 = ashr i32 %89, 24
	%91 = trunc i32 %90 to i8
	store i8 %91, i8* %86
	%92 = load i32, i32* %13
	%93 = add i32 %92, 1
	store i32 %93, i32* %13
	br label %94

94:
	%95 = load i32, i32* %13
	%96 = load i32*, i32** %6
	%97 = getelementptr i8, i32* %96, i32 %95
	%98 = load i32, i32* %21
	%99 = shl i32 %98, 24
	%100 = ashr i32 %99, 24
	%101 = shl i32 %100, 24
	%102 = ashr i32 %101, 24
	%103 = shl i32 %102, 24
	%104 = ashr i32 %103, 24
	%105 = trunc i32 %104 to i8
	store i8 %105, i8* %97
	%106 = load i32, i32* %13
	%107 = add i32 %106, 1
	store i32 %107, i32* %13
	%108 = load i32, i32* %12
	%109 = add i32 %108, 1
	store i32 %109, i32* %12
	br label %14

dead120:
	ret i32* null
}

define i32* @rt_catfields(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	%5 = load i32*, i32** %3
	%6 = bitcast i32* %5 to i32*
	%7 = call i32 @rt_nfields(i32* %6)
	store i32 %7, i32* %4
	%8 = alloca i32*
	%9 = load i32*, i32** %2
	%10 = bitcast i32* %9 to i32*
	store i32* %10, i32** %8
	%11 = alloca i32
	store i32 0, i32* %11
	br label %12

12:
	%13 = load i32, i32* %11
	%14 = load i32, i32* %4
	%15 = icmp slt i32 %13, %14
	%16 = zext i1 %15 to i32
	%17 = icmp ne i32 %16, 0
	br i1 %17, label %18, label %42

18:
	%19 = alloca i32*
	%20 = load i32*, i32** %3
	%21 = bitcast i32* %20 to i32*
	%22 = load i32, i32* %11
	%23 = call i32* @rt_getfield(i32* %21, i32 %22)
	%24 = bitcast i32* %23 to i32*
	store i32* %24, i32** %19
	%25 = alloca i32*
	%26 = load i32*, i32** %8
	%27 = bitcast i32* %26 to i32*
	%28 = load i32*, i32** %19
	%29 = bitcast i32* %28 to i32*
	%30 = call i32* @rt_strcat(i32* %27, i32* %29)
	%31 = bitcast i32* %30 to i32*
	store i32* %31, i32** %25
	%32 = load i32*, i32** %25
	%33 = bitcast i32* %32 to i32*
	%34 = getelementptr [2 x i8], [2 x i8]* @.str.28, i32 0, i32 0
	store i8 1, i8* %34
	%35 = getelementptr [2 x i8], [2 x i8]* @.str.28, i32 0, i32 1
	store i8 0, i8* %35
	%36 = getelementptr [2 x i8], [2 x i8]* @.str.28, i32 0, i32 0
	%37 = bitcast i8* %36 to i32*
	%38 = call i32* @rt_strcat(i32* %33, i32* %37)
	%39 = bitcast i32* %38 to i32*
	store i32* %39, i32** %8
	%40 = load i32, i32* %11
	%41 = add i32 %40, 1
	store i32 %41, i32* %11
	br label %12

42:
	%43 = load i32*, i32** %8
	%44 = bitcast i32* %43 to i32*
	ret i32* %44

dead121:
	ret i32* null
}

define i32 @rt_argpush(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = call i32 @rt_nfields(i32* %4)
	store i32 %5, i32* %2
	%6 = alloca i32
	store i32 0, i32* %6
	br label %7

7:
	%8 = load i32, i32* %6
	%9 = load i32, i32* %2
	%10 = icmp slt i32 %8, %9
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %13, label %20

13:
	%14 = alloca i32
	%15 = load i32, i32* @argv_top
	store i32 %15, i32* %14
	%16 = load i32, i32* %14
	%17 = icmp sge i32 %16, u0x1000
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %21, label %22

20:
	ret i32 0

21:
	store i32 1, i32* @rt_limit
	ret i32 0

22:
	%23 = load i32, i32* %14
	%24 = getelementptr [4096 x i32*], [4096 x i32*]* @argv, i32 0, i32 %23
	%25 = load i32*, i32** %1
	%26 = bitcast i32* %25 to i32*
	%27 = load i32, i32* %6
	%28 = call i32* @rt_getfield(i32* %26, i32 %27)
	%29 = bitcast i32* %28 to i32*
	store i32* %29, i32** %24
	%30 = load i32, i32* %14
	%31 = add i32 %30, 1
	store i32 %31, i32* @argv_top
	%32 = load i32, i32* %6
	%33 = add i32 %32, 1
	store i32 %33, i32* %6
	br label %7

dead122:
	br label %22

dead123:
	ret i32 0
}

define i32* @rt_param(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = load i32, i32* @frame_n
	store i32 %3, i32* %2
	%4 = load i32, i32* %1
	%5 = icmp sge i32 %4, 1
	%6 = zext i1 %5 to i32
	%7 = icmp ne i32 %6, 0
	%8 = zext i1 %7 to i32
	%9 = icmp ne i32 %8, 0
	br i1 %9, label %10, label %17

10:
	%11 = load i32, i32* %1
	%12 = load i32, i32* %2
	%13 = icmp sle i32 %11, %12
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	%16 = zext i1 %15 to i32
	br label %17

17:
	%18 = phi i32 [ %8, %entry ], [ %16, %10 ]
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %30

20:
	%21 = alloca i32
	%22 = load i32, i32* @frame_base
	%23 = load i32, i32* %1
	%24 = sub i32 %23, 1
	%25 = add i32 %22, %24
	store i32 %25, i32* %21
	%26 = load i32, i32* %21
	%27 = getelementptr [4096 x i32*], [4096 x i32*]* @argv, i32 0, i32 %26
	%28 = load i32*, i32** %27
	%29 = bitcast i32* %28 to i32*
	ret i32* %29

30:
	%31 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%32 = bitcast i8* %31 to i32*
	ret i32* %32

dead124:
	br label %30

dead125:
	ret i32* null
}

define i32* @rt_params(i32* %0, i32 %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = load i32, i32* @frame_n
	store i32 %5, i32* %4
	%6 = alloca i32
	%7 = load i32, i32* @frame_base
	store i32 %7, i32* %6
	%8 = alloca i32
	%9 = load i32, i32* %3
	%10 = icmp ne i32 %9, 0
	%11 = zext i1 %10 to i32
	store i32 %11, i32* %8
	%12 = alloca i32*
	%13 = load i32, i32* %8
	%14 = icmp ne i32 %13, 0
	br i1 %14, label %15, label %18

15:
	%16 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%17 = bitcast i8* %16 to i32*
	br label %23

18:
	%19 = getelementptr [2 x i8], [2 x i8]* @.str.29, i32 0, i32 0
	store i8 2, i8* %19
	%20 = getelementptr [2 x i8], [2 x i8]* @.str.29, i32 0, i32 1
	store i8 0, i8* %20
	%21 = getelementptr [2 x i8], [2 x i8]* @.str.29, i32 0, i32 0
	%22 = bitcast i8* %21 to i32*
	br label %23

23:
	%24 = phi i32* [ %17, %15 ], [ %22, %18 ]
	%25 = bitcast i32* %24 to i32*
	store i32* %25, i32** %12
	%26 = alloca i32
	store i32 0, i32* %26
	br label %27

27:
	%28 = load i32, i32* %26
	%29 = load i32, i32* %4
	%30 = icmp slt i32 %28, %29
	%31 = zext i1 %30 to i32
	%32 = icmp ne i32 %31, 0
	br i1 %32, label %33, label %49

33:
	%34 = alloca i32*
	%35 = load i32, i32* %6
	%36 = load i32, i32* %26
	%37 = add i32 %35, %36
	%38 = getelementptr [4096 x i32*], [4096 x i32*]* @argv, i32 0, i32 %37
	%39 = load i32*, i32** %38
	%40 = bitcast i32* %39 to i32*
	store i32* %40, i32** %34
	%41 = alloca i32*
	%42 = load i32*, i32** %12
	%43 = bitcast i32* %42 to i32*
	store i32* %43, i32** %41
	%44 = alloca i32
	%45 = load i32, i32* %8
	%46 = icmp ne i32 %45, 0
	%47 = zext i1 %46 to i32
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %52, label %58

49:
	%50 = load i32*, i32** %12
	%51 = bitcast i32* %50 to i32*
	ret i32* %51

52:
	%53 = load i32, i32* %26
	%54 = icmp sgt i32 %53, 0
	%55 = zext i1 %54 to i32
	%56 = icmp ne i32 %55, 0
	%57 = zext i1 %56 to i32
	br label %58

58:
	%59 = phi i32 [ %47, %33 ], [ %57, %52 ]
	store i32 %59, i32* %44
	%60 = alloca i32*
	%61 = load i32*, i32** %41
	%62 = bitcast i32* %61 to i32*
	%63 = load i32*, i32** %2
	%64 = bitcast i32* %63 to i32*
	%65 = call i32* @rt_strcat(i32* %62, i32* %64)
	%66 = bitcast i32* %65 to i32*
	store i32* %66, i32** %60
	%67 = load i32, i32* %44
	%68 = icmp ne i32 %67, 0
	br i1 %68, label %69, label %72

69:
	%70 = load i32*, i32** %60
	%71 = bitcast i32* %70 to i32*
	store i32* %71, i32** %41
	br label %72

72:
	%73 = load i32*, i32** %41
	%74 = bitcast i32* %73 to i32*
	%75 = load i32*, i32** %34
	%76 = bitcast i32* %75 to i32*
	%77 = call i32* @rt_strcat(i32* %74, i32* %76)
	%78 = bitcast i32* %77 to i32*
	store i32* %78, i32** %41
	%79 = alloca i32*
	%80 = load i32*, i32** %41
	%81 = bitcast i32* %80 to i32*
	%82 = getelementptr [2 x i8], [2 x i8]* @.str.30, i32 0, i32 0
	store i8 1, i8* %82
	%83 = getelementptr [2 x i8], [2 x i8]* @.str.30, i32 0, i32 1
	store i8 0, i8* %83
	%84 = getelementptr [2 x i8], [2 x i8]* @.str.30, i32 0, i32 0
	%85 = bitcast i8* %84 to i32*
	%86 = call i32* @rt_strcat(i32* %81, i32* %85)
	%87 = bitcast i32* %86 to i32*
	store i32* %87, i32** %79
	%88 = load i32, i32* %8
	%89 = icmp eq i32 %88, 0
	%90 = zext i1 %89 to i32
	%91 = icmp ne i32 %90, 0
	br i1 %91, label %92, label %95

92:
	%93 = load i32*, i32** %79
	%94 = bitcast i32* %93 to i32*
	store i32* %94, i32** %41
	br label %95

95:
	%96 = load i32*, i32** %41
	%97 = bitcast i32* %96 to i32*
	store i32* %97, i32** %12
	%98 = load i32, i32* %26
	%99 = add i32 %98, 1
	store i32 %99, i32* %26
	br label %27

dead126:
	ret i32* null
}

define i32* @rt_nounset(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32*
	%3 = getelementptr [1 x i8], [1 x i8]* @unset_marker, i32 0, i32 0
	%4 = bitcast i8* %3 to i32*
	store i32* %4, i32** %2
	%5 = load i32, i32* @opt_nounset
	%6 = icmp ne i32 %5, 0
	%7 = zext i1 %6 to i32
	%8 = icmp ne i32 %7, 0
	%9 = zext i1 %8 to i32
	%10 = icmp ne i32 %9, 0
	br i1 %10, label %11, label %20

11:
	%12 = load i32*, i32** %1
	%13 = load i32*, i32** %2
	%14 = bitcast i32* %12 to i32*
	%15 = bitcast i32* %13 to i32*
	%16 = icmp eq i32* %14, %15
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	%19 = zext i1 %18 to i32
	br label %20

20:
	%21 = phi i32 [ %9, %entry ], [ %19, %11 ]
	%22 = icmp ne i32 %21, 0
	br i1 %22, label %23, label %24

23:
	store i32 1, i32* @abort_flag
	br label %24

24:
	%25 = load i32*, i32** %1
	%26 = bitcast i32* %25 to i32*
	ret i32* %26

dead127:
	ret i32* null
}

define i32 @rt_putc(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32*
	%5 = alloca i32
	%6 = load i32, i32* @sink_out
	store i32 %6, i32* %2
	%7 = load i32, i32* %2
	%8 = icmp eq i32 %7, 0
	%9 = zext i1 %8 to i32
	%10 = icmp ne i32 %9, 0
	br i1 %10, label %11, label %17

11:
	%12 = load i32, i32* @cap_depth
	store i32 %12, i32* %3
	%13 = load i32, i32* %3
	%14 = icmp sgt i32 %13, 0
	%15 = zext i1 %14 to i32
	%16 = icmp ne i32 %15, 0
	br i1 %16, label %18, label %38

17:
	ret i32 0

18:
	%19 = load i32, i32* %3
	%20 = sub i32 %19, 1
	%21 = getelementptr [16 x i32*], [16 x i32*]* @cap_buf, i32 0, i32 %20
	%22 = load i32*, i32** %21
	%23 = bitcast i32* %22 to i32*
	store i32* %23, i32** %4
	%24 = load i32, i32* %3
	%25 = sub i32 %24, 1
	%26 = getelementptr [16 x i32], [16 x i32]* @cap_len, i32 0, i32 %25
	%27 = load i32, i32* %26
	store i32 %27, i32* %5
	%28 = load i32, i32* %5
	%29 = add i32 %28, 1
	%30 = load i32, i32* %3
	%31 = sub i32 %30, 1
	%32 = getelementptr [16 x i32], [16 x i32]* @cap_sz, i32 0, i32 %31
	%33 = load i32, i32* %32
	%34 = icmp sge i32 %29, %33
	%35 = zext i1 %34 to i32
	%36 = icmp ne i32 %35, 0
	br i1 %36, label %41, label %53

37:
	br label %17

38:
	%39 = load i32, i32* %1
	%40 = call i32 @putchar(i32 %39)
	br label %37

41:
	%42 = alloca i32
	%43 = alloca i32*
	%44 = alloca i32
	%45 = load i32, i32* %3
	%46 = sub i32 %45, 1
	%47 = getelementptr [16 x i32], [16 x i32]* @cap_sz, i32 0, i32 %46
	%48 = load i32, i32* %47
	%49 = mul i32 %48, 2
	store i32 %49, i32* %42
	%50 = load i32, i32* %42
	%51 = call i32* @rt_bump(i32 %50)
	%52 = bitcast i32* %51 to i32*
	store i32* %52, i32** %43
	store i32 0, i32* %44
	br label %70

53:
	%54 = load i32, i32* %5
	%55 = load i32*, i32** %4
	%56 = getelementptr i8, i32* %55, i32 %54
	%57 = load i32, i32* %1
	%58 = shl i32 %57, 24
	%59 = ashr i32 %58, 24
	%60 = shl i32 %59, 24
	%61 = ashr i32 %60, 24
	%62 = shl i32 %61, 24
	%63 = ashr i32 %62, 24
	%64 = trunc i32 %63 to i8
	store i8 %64, i8* %56
	%65 = load i32, i32* %3
	%66 = sub i32 %65, 1
	%67 = getelementptr [16 x i32], [16 x i32]* @cap_len, i32 0, i32 %66
	%68 = load i32, i32* %5
	%69 = add i32 %68, 1
	store i32 %69, i32* %67
	br label %37

70:
	%71 = load i32, i32* %44
	%72 = load i32, i32* %5
	%73 = icmp slt i32 %71, %72
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	br i1 %75, label %76, label %92

76:
	%77 = load i32, i32* %44
	%78 = load i32*, i32** %43
	%79 = getelementptr i8, i32* %78, i32 %77
	%80 = load i32, i32* %44
	%81 = load i32*, i32** %4
	%82 = getelementptr i8, i32* %81, i32 %80
	%83 = load i8, i8* %82
	%84 = sext i8 %83 to i32
	%85 = shl i32 %84, 24
	%86 = ashr i32 %85, 24
	%87 = shl i32 %86, 24
	%88 = ashr i32 %87, 24
	%89 = trunc i32 %88 to i8
	store i8 %89, i8* %79
	%90 = load i32, i32* %44
	%91 = add i32 %90, 1
	store i32 %91, i32* %44
	br label %70

92:
	%93 = load i32, i32* %3
	%94 = sub i32 %93, 1
	%95 = getelementptr [16 x i32*], [16 x i32*]* @cap_buf, i32 0, i32 %94
	%96 = load i32*, i32** %43
	%97 = bitcast i32* %96 to i32*
	store i32* %97, i32** %95
	%98 = load i32, i32* %3
	%99 = sub i32 %98, 1
	%100 = getelementptr [16 x i32], [16 x i32]* @cap_sz, i32 0, i32 %99
	%101 = load i32, i32* %42
	store i32 %101, i32* %100
	%102 = load i32*, i32** %43
	%103 = bitcast i32* %102 to i32*
	store i32* %103, i32** %4
	br label %53

dead128:
	ret i32 0
}

declare i32 @putchar(i32 %0)

define i32 @rt_cap_begin(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32*
	%4 = load i32, i32* @cap_depth
	store i32 %4, i32* %2
	%5 = load i32, i32* %2
	%6 = icmp sge i32 %5, 16
	%7 = zext i1 %6 to i32
	%8 = icmp ne i32 %7, 0
	br i1 %8, label %9, label %10

9:
	store i32 1, i32* @rt_limit
	ret i32 0

10:
	%11 = call i32* @rt_bump(i32 u0x2000)
	%12 = bitcast i32* %11 to i32*
	store i32* %12, i32** %3
	%13 = load i32, i32* %2
	%14 = getelementptr [16 x i32*], [16 x i32*]* @cap_buf, i32 0, i32 %13
	%15 = load i32*, i32** %3
	%16 = bitcast i32* %15 to i32*
	store i32* %16, i32** %14
	%17 = load i32, i32* %2
	%18 = getelementptr [16 x i32], [16 x i32]* @cap_sz, i32 0, i32 %17
	store i32 u0x2000, i32* %18
	%19 = load i32, i32* %2
	%20 = getelementptr [16 x i32], [16 x i32]* @cap_len, i32 0, i32 %19
	store i32 0, i32* %20
	%21 = load i32, i32* %2
	%22 = add i32 %21, 1
	store i32 %22, i32* @cap_depth
	ret i32 0

dead129:
	br label %10

dead130:
	ret i32 0
}

define i32* @rt_cap_end(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32*
	%4 = alloca i32
	%5 = load i32, i32* @cap_depth
	%6 = sub i32 %5, 1
	store i32 %6, i32* %2
	%7 = load i32, i32* %2
	store i32 %7, i32* @cap_depth
	%8 = load i32, i32* %2
	%9 = getelementptr [16 x i32*], [16 x i32*]* @cap_buf, i32 0, i32 %8
	%10 = load i32*, i32** %9
	%11 = bitcast i32* %10 to i32*
	store i32* %11, i32** %3
	%12 = load i32, i32* %2
	%13 = getelementptr [16 x i32], [16 x i32]* @cap_len, i32 0, i32 %12
	%14 = load i32, i32* %13
	store i32 %14, i32* %4
	br label %15

15:
	%16 = load i32, i32* %4
	%17 = icmp sgt i32 %16, 0
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	br i1 %19, label %20, label %30

20:
	%21 = load i32, i32* %4
	%22 = sub i32 %21, 1
	%23 = load i32*, i32** %3
	%24 = getelementptr i8, i32* %23, i32 %22
	%25 = load i8, i8* %24
	%26 = sext i8 %25 to i32
	%27 = icmp ne i32 %26, 10
	%28 = zext i1 %27 to i32
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %41, label %42

30:
	%31 = load i32, i32* %4
	%32 = load i32*, i32** %3
	%33 = getelementptr i8, i32* %32, i32 %31
	%34 = shl i32 0, 24
	%35 = ashr i32 %34, 24
	%36 = shl i32 %35, 24
	%37 = ashr i32 %36, 24
	%38 = trunc i32 %37 to i8
	store i8 %38, i8* %33
	%39 = load i32*, i32** %3
	%40 = bitcast i32* %39 to i32*
	ret i32* %40

41:
	br label %30

42:
	%43 = load i32, i32* %4
	%44 = sub i32 %43, 1
	store i32 %44, i32* %4
	br label %15

dead131:
	br label %42

dead132:
	ret i32* null
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
	%19 = call i32 @rt_putc(i32 %18)
	%20 = load i32, i32* %2
	%21 = add i32 %20, 1
	store i32 %21, i32* %2
	br label %3

22:
	ret i32 0

dead133:
	ret i32 0
}

define i32 @rt_ss_save(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	%5 = load i32, i32* @ss_depth
	store i32 %5, i32* %2
	%6 = load i32, i32* %2
	%7 = mul i32 %6, 1024
	store i32 %7, i32* %3
	store i32 0, i32* %4
	br label %8

8:
	%9 = load i32, i32* %4
	%10 = load i32, i32* @nvars
	%11 = icmp slt i32 %9, %10
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	br i1 %13, label %14, label %25

14:
	%15 = load i32, i32* %3
	%16 = load i32, i32* %4
	%17 = add i32 %15, %16
	%18 = getelementptr [8192 x i32*], [8192 x i32*]* @ss_save, i32 0, i32 %17
	%19 = load i32, i32* %4
	%20 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %19
	%21 = load i32*, i32** %20
	%22 = bitcast i32* %21 to i32*
	store i32* %22, i32** %18
	%23 = load i32, i32* %4
	%24 = add i32 %23, 1
	store i32 %24, i32* %4
	br label %8

25:
	%26 = load i32, i32* %2
	%27 = add i32 %26, 1
	store i32 %27, i32* @ss_depth
	ret i32 0

dead134:
	ret i32 0
}

define i32 @rt_ss_restore(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	%5 = load i32, i32* @ss_depth
	%6 = sub i32 %5, 1
	store i32 %6, i32* %2
	%7 = load i32, i32* %2
	store i32 %7, i32* @ss_depth
	%8 = load i32, i32* %2
	%9 = mul i32 %8, 1024
	store i32 %9, i32* %3
	store i32 0, i32* %4
	br label %10

10:
	%11 = load i32, i32* %4
	%12 = load i32, i32* @nvars
	%13 = icmp slt i32 %11, %12
	%14 = zext i1 %13 to i32
	%15 = icmp ne i32 %14, 0
	br i1 %15, label %16, label %27

16:
	%17 = load i32, i32* %4
	%18 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %17
	%19 = load i32, i32* %3
	%20 = load i32, i32* %4
	%21 = add i32 %19, %20
	%22 = getelementptr [8192 x i32*], [8192 x i32*]* @ss_save, i32 0, i32 %21
	%23 = load i32*, i32** %22
	%24 = bitcast i32* %23 to i32*
	store i32* %24, i32** %18
	%25 = load i32, i32* %4
	%26 = add i32 %25, 1
	store i32 %26, i32* %4
	br label %10

27:
	ret i32 0

dead135:
	ret i32 0
}

define i32 @rt_push_local(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = load i32, i32* @ls_top
	store i32 %3, i32* %2
	%4 = load i32, i32* %2
	%5 = icmp sge i32 %4, 512
	%6 = zext i1 %5 to i32
	%7 = icmp ne i32 %6, 0
	br i1 %7, label %8, label %9

8:
	store i32 1, i32* @rt_limit
	ret i32 0

9:
	%10 = load i32, i32* %2
	%11 = getelementptr [512 x i32], [512 x i32]* @ls_id, i32 0, i32 %10
	%12 = load i32, i32* %1
	store i32 %12, i32* %11
	%13 = load i32, i32* %2
	%14 = getelementptr [512 x i32*], [512 x i32*]* @ls_val, i32 0, i32 %13
	%15 = load i32, i32* %1
	%16 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %15
	%17 = load i32*, i32** %16
	%18 = bitcast i32* %17 to i32*
	store i32* %18, i32** %14
	%19 = load i32, i32* %2
	%20 = add i32 %19, 1
	store i32 %20, i32* @ls_top
	ret i32 0

dead136:
	br label %9

dead137:
	ret i32 0
}

define i32 @rt_pop_locals(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = load i32, i32* @ls_top
	store i32 %3, i32* %2
	br label %4

4:
	%5 = load i32, i32* %2
	%6 = load i32, i32* %1
	%7 = icmp sgt i32 %5, %6
	%8 = zext i1 %7 to i32
	%9 = icmp ne i32 %8, 0
	br i1 %9, label %10, label %23

10:
	%11 = load i32, i32* %2
	%12 = sub i32 %11, 1
	store i32 %12, i32* %2
	%13 = load i32, i32* %2
	store i32 %13, i32* @ls_top
	%14 = load i32, i32* %2
	%15 = getelementptr [512 x i32], [512 x i32]* @ls_id, i32 0, i32 %14
	%16 = load i32, i32* %15
	%17 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %16
	%18 = load i32, i32* %2
	%19 = getelementptr [512 x i32*], [512 x i32*]* @ls_val, i32 0, i32 %18
	%20 = load i32*, i32** %19
	%21 = bitcast i32* %20 to i32*
	store i32* %21, i32** %17
	%22 = load i32, i32* @ls_top
	store i32 %22, i32* %2
	br label %4

23:
	ret i32 0

dead138:
	ret i32 0
}

define i32* @rt_getvar_byname(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32*
	%3 = alloca i32
	%4 = getelementptr [1 x i8], [1 x i8]* @unset_marker, i32 0, i32 0
	%5 = bitcast i8* %4 to i32*
	store i32* %5, i32** %2
	store i32 0, i32* %3
	br label %6

6:
	%7 = load i32, i32* %3
	%8 = load i32, i32* @nvars
	%9 = icmp slt i32 %7, %8
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %23

12:
	%13 = load i32*, i32** %1
	%14 = bitcast i32* %13 to i32*
	%15 = load i32, i32* %3
	%16 = getelementptr [1024 x i32*], [1024 x i32*]* @varnames, i32 0, i32 %15
	%17 = load i32*, i32** %16
	%18 = bitcast i32* %17 to i32*
	%19 = call i32 @rt_streq(i32* %14, i32* %18)
	%20 = icmp ne i32 %19, 0
	%21 = zext i1 %20 to i32
	%22 = icmp ne i32 %21, 0
	br i1 %22, label %26, label %31

23:
	%24 = load i32*, i32** %2
	%25 = bitcast i32* %24 to i32*
	ret i32* %25

26:
	%27 = load i32, i32* %3
	%28 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %27
	%29 = load i32*, i32** %28
	%30 = bitcast i32* %29 to i32*
	store i32* %30, i32** %2
	br label %31

31:
	%32 = load i32, i32* %3
	%33 = add i32 %32, 1
	store i32 %33, i32* %3
	br label %6

dead139:
	ret i32* null
}

define i32 @rt_setvar_byname(i32* %0, i32* %1) {
entry:
	%2 = alloca i32*
	store i32* %0, i32** %2
	%3 = alloca i32*
	store i32* %1, i32** %3
	%4 = alloca i32
	store i32 0, i32* %4
	br label %5

5:
	%6 = load i32, i32* %4
	%7 = load i32, i32* @nvars
	%8 = icmp slt i32 %6, %7
	%9 = zext i1 %8 to i32
	%10 = icmp ne i32 %9, 0
	br i1 %10, label %11, label %22

11:
	%12 = load i32*, i32** %2
	%13 = bitcast i32* %12 to i32*
	%14 = load i32, i32* %4
	%15 = getelementptr [1024 x i32*], [1024 x i32*]* @varnames, i32 0, i32 %14
	%16 = load i32*, i32** %15
	%17 = bitcast i32* %16 to i32*
	%18 = call i32 @rt_streq(i32* %13, i32* %17)
	%19 = icmp ne i32 %18, 0
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	br i1 %21, label %23, label %28

22:
	ret i32 0

23:
	%24 = load i32, i32* %4
	%25 = getelementptr [1024 x i32*], [1024 x i32*]* @gvars, i32 0, i32 %24
	%26 = load i32*, i32** %3
	%27 = bitcast i32* %26 to i32*
	store i32* %27, i32** %25
	br label %28

28:
	%29 = load i32, i32* %4
	%30 = add i32 %29, 1
	store i32 %30, i32* %4
	br label %5

dead140:
	ret i32 0
}

define i32 @rt_eval_assign(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = load i32*, i32** %1
	%5 = bitcast i32* %4 to i32*
	%6 = call i32 @rt_strlen(i32* %5)
	store i32 %6, i32* %2
	store i32 0, i32* %3
	br label %7

7:
	%8 = load i32, i32* %3
	%9 = load i32, i32* %2
	%10 = icmp slt i32 %8, %9
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %13, label %22

13:
	%14 = load i32, i32* %3
	%15 = load i32*, i32** %1
	%16 = getelementptr i8, i32* %15, i32 %14
	%17 = load i8, i8* %16
	%18 = sext i8 %17 to i32
	%19 = icmp eq i32 %18, 61
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	br i1 %21, label %23, label %36

22:
	ret i32 0

23:
	%24 = load i32*, i32** %1
	%25 = bitcast i32* %24 to i32*
	%26 = load i32, i32* %3
	%27 = call i32* @rt_substr(i32* %25, i32 0, i32 %26)
	%28 = bitcast i32* %27 to i32*
	%29 = load i32*, i32** %1
	%30 = bitcast i32* %29 to i32*
	%31 = load i32, i32* %3
	%32 = add i32 %31, 1
	%33 = call i32* @rt_substr(i32* %30, i32 %32, i32 -1)
	%34 = bitcast i32* %33 to i32*
	%35 = call i32 @rt_setvar_byname(i32* %28, i32* %34)
	ret i32 0

36:
	%37 = load i32, i32* %3
	%38 = add i32 %37, 1
	store i32 %38, i32* %3
	br label %7

dead141:
	br label %36

dead142:
	ret i32 0
}

define i32 @rt_fskind(i32* %0) {
entry:
	%1 = alloca i32*
	store i32* %0, i32** %1
	%2 = alloca i32
	store i32 0, i32* %2
	%3 = load i32*, i32** %1
	%4 = bitcast i32* %3 to i32*
	%5 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 0
	store i8 47, i8* %5
	%6 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 1
	store i8 100, i8* %6
	%7 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 2
	store i8 101, i8* %7
	%8 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 3
	store i8 118, i8* %8
	%9 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 4
	store i8 47, i8* %9
	%10 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 5
	store i8 110, i8* %10
	%11 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 6
	store i8 117, i8* %11
	%12 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 7
	store i8 108, i8* %12
	%13 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 8
	store i8 108, i8* %13
	%14 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 9
	store i8 0, i8* %14
	%15 = getelementptr [10 x i8], [10 x i8]* @.str.31, i32 0, i32 0
	%16 = bitcast i8* %15 to i32*
	%17 = call i32 @rt_streq(i32* %4, i32* %16)
	%18 = icmp ne i32 %17, 0
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	br i1 %20, label %21, label %22

21:
	br label %24

22:
	%23 = load i32, i32* %2
	br label %24

24:
	%25 = phi i32 [ 1, %21 ], [ %23, %22 ]
	store i32 %25, i32* %2
	%26 = load i32*, i32** %1
	%27 = bitcast i32* %26 to i32*
	%28 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 0
	store i8 47, i8* %28
	%29 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 1
	store i8 100, i8* %29
	%30 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 2
	store i8 101, i8* %30
	%31 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 3
	store i8 118, i8* %31
	%32 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 4
	store i8 47, i8* %32
	%33 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 5
	store i8 122, i8* %33
	%34 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 6
	store i8 101, i8* %34
	%35 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 7
	store i8 114, i8* %35
	%36 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 8
	store i8 111, i8* %36
	%37 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 9
	store i8 0, i8* %37
	%38 = getelementptr [10 x i8], [10 x i8]* @.str.32, i32 0, i32 0
	%39 = bitcast i8* %38 to i32*
	%40 = call i32 @rt_streq(i32* %27, i32* %39)
	%41 = icmp ne i32 %40, 0
	%42 = zext i1 %41 to i32
	%43 = icmp ne i32 %42, 0
	br i1 %43, label %44, label %45

44:
	br label %47

45:
	%46 = load i32, i32* %2
	br label %47

47:
	%48 = phi i32 [ 1, %44 ], [ %46, %45 ]
	store i32 %48, i32* %2
	%49 = load i32*, i32** %1
	%50 = bitcast i32* %49 to i32*
	%51 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 0
	store i8 47, i8* %51
	%52 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 1
	store i8 100, i8* %52
	%53 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 2
	store i8 101, i8* %53
	%54 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 3
	store i8 118, i8* %54
	%55 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 4
	store i8 47, i8* %55
	%56 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 5
	store i8 116, i8* %56
	%57 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 6
	store i8 116, i8* %57
	%58 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 7
	store i8 121, i8* %58
	%59 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 8
	store i8 0, i8* %59
	%60 = getelementptr [9 x i8], [9 x i8]* @.str.33, i32 0, i32 0
	%61 = bitcast i8* %60 to i32*
	%62 = call i32 @rt_streq(i32* %50, i32* %61)
	%63 = icmp ne i32 %62, 0
	%64 = zext i1 %63 to i32
	%65 = icmp ne i32 %64, 0
	br i1 %65, label %66, label %67

66:
	br label %69

67:
	%68 = load i32, i32* %2
	br label %69

69:
	%70 = phi i32 [ 1, %66 ], [ %68, %67 ]
	store i32 %70, i32* %2
	%71 = load i32*, i32** %1
	%72 = bitcast i32* %71 to i32*
	%73 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 0
	store i8 47, i8* %73
	%74 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 1
	store i8 100, i8* %74
	%75 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 2
	store i8 101, i8* %75
	%76 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 3
	store i8 118, i8* %76
	%77 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 4
	store i8 47, i8* %77
	%78 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 5
	store i8 115, i8* %78
	%79 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 6
	store i8 116, i8* %79
	%80 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 7
	store i8 100, i8* %80
	%81 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 8
	store i8 105, i8* %81
	%82 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 9
	store i8 110, i8* %82
	%83 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 10
	store i8 0, i8* %83
	%84 = getelementptr [11 x i8], [11 x i8]* @.str.34, i32 0, i32 0
	%85 = bitcast i8* %84 to i32*
	%86 = call i32 @rt_streq(i32* %72, i32* %85)
	%87 = icmp ne i32 %86, 0
	%88 = zext i1 %87 to i32
	%89 = icmp ne i32 %88, 0
	br i1 %89, label %90, label %91

90:
	br label %93

91:
	%92 = load i32, i32* %2
	br label %93

93:
	%94 = phi i32 [ 1, %90 ], [ %92, %91 ]
	store i32 %94, i32* %2
	%95 = load i32*, i32** %1
	%96 = bitcast i32* %95 to i32*
	%97 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 0
	store i8 47, i8* %97
	%98 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 1
	store i8 100, i8* %98
	%99 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 2
	store i8 101, i8* %99
	%100 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 3
	store i8 118, i8* %100
	%101 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 4
	store i8 47, i8* %101
	%102 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 5
	store i8 115, i8* %102
	%103 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 6
	store i8 116, i8* %103
	%104 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 7
	store i8 100, i8* %104
	%105 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 8
	store i8 111, i8* %105
	%106 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 9
	store i8 117, i8* %106
	%107 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 10
	store i8 116, i8* %107
	%108 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 11
	store i8 0, i8* %108
	%109 = getelementptr [12 x i8], [12 x i8]* @.str.35, i32 0, i32 0
	%110 = bitcast i8* %109 to i32*
	%111 = call i32 @rt_streq(i32* %96, i32* %110)
	%112 = icmp ne i32 %111, 0
	%113 = zext i1 %112 to i32
	%114 = icmp ne i32 %113, 0
	br i1 %114, label %115, label %116

115:
	br label %118

116:
	%117 = load i32, i32* %2
	br label %118

118:
	%119 = phi i32 [ 1, %115 ], [ %117, %116 ]
	store i32 %119, i32* %2
	%120 = load i32*, i32** %1
	%121 = bitcast i32* %120 to i32*
	%122 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 0
	store i8 47, i8* %122
	%123 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 1
	store i8 100, i8* %123
	%124 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 2
	store i8 101, i8* %124
	%125 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 3
	store i8 118, i8* %125
	%126 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 4
	store i8 47, i8* %126
	%127 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 5
	store i8 115, i8* %127
	%128 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 6
	store i8 116, i8* %128
	%129 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 7
	store i8 100, i8* %129
	%130 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 8
	store i8 101, i8* %130
	%131 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 9
	store i8 114, i8* %131
	%132 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 10
	store i8 114, i8* %132
	%133 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 11
	store i8 0, i8* %133
	%134 = getelementptr [12 x i8], [12 x i8]* @.str.36, i32 0, i32 0
	%135 = bitcast i8* %134 to i32*
	%136 = call i32 @rt_streq(i32* %121, i32* %135)
	%137 = icmp ne i32 %136, 0
	%138 = zext i1 %137 to i32
	%139 = icmp ne i32 %138, 0
	br i1 %139, label %140, label %141

140:
	br label %143

141:
	%142 = load i32, i32* %2
	br label %143

143:
	%144 = phi i32 [ 1, %140 ], [ %142, %141 ]
	store i32 %144, i32* %2
	%145 = load i32*, i32** %1
	%146 = bitcast i32* %145 to i32*
	%147 = getelementptr [2 x i8], [2 x i8]* @.str.37, i32 0, i32 0
	store i8 47, i8* %147
	%148 = getelementptr [2 x i8], [2 x i8]* @.str.37, i32 0, i32 1
	store i8 0, i8* %148
	%149 = getelementptr [2 x i8], [2 x i8]* @.str.37, i32 0, i32 0
	%150 = bitcast i8* %149 to i32*
	%151 = call i32 @rt_streq(i32* %146, i32* %150)
	%152 = icmp ne i32 %151, 0
	%153 = zext i1 %152 to i32
	%154 = icmp ne i32 %153, 0
	br i1 %154, label %155, label %156

155:
	br label %158

156:
	%157 = load i32, i32* %2
	br label %158

158:
	%159 = phi i32 [ 2, %155 ], [ %157, %156 ]
	store i32 %159, i32* %2
	%160 = load i32*, i32** %1
	%161 = bitcast i32* %160 to i32*
	%162 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 0
	store i8 47, i8* %162
	%163 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 1
	store i8 100, i8* %163
	%164 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 2
	store i8 101, i8* %164
	%165 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 3
	store i8 118, i8* %165
	%166 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 4
	store i8 0, i8* %166
	%167 = getelementptr [5 x i8], [5 x i8]* @.str.38, i32 0, i32 0
	%168 = bitcast i8* %167 to i32*
	%169 = call i32 @rt_streq(i32* %161, i32* %168)
	%170 = icmp ne i32 %169, 0
	%171 = zext i1 %170 to i32
	%172 = icmp ne i32 %171, 0
	br i1 %172, label %173, label %174

173:
	br label %176

174:
	%175 = load i32, i32* %2
	br label %176

176:
	%177 = phi i32 [ 2, %173 ], [ %175, %174 ]
	store i32 %177, i32* %2
	%178 = load i32*, i32** %1
	%179 = bitcast i32* %178 to i32*
	%180 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 0
	store i8 47, i8* %180
	%181 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 1
	store i8 116, i8* %181
	%182 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 2
	store i8 109, i8* %182
	%183 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 3
	store i8 112, i8* %183
	%184 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 4
	store i8 0, i8* %184
	%185 = getelementptr [5 x i8], [5 x i8]* @.str.39, i32 0, i32 0
	%186 = bitcast i8* %185 to i32*
	%187 = call i32 @rt_streq(i32* %179, i32* %186)
	%188 = icmp ne i32 %187, 0
	%189 = zext i1 %188 to i32
	%190 = icmp ne i32 %189, 0
	br i1 %190, label %191, label %192

191:
	br label %194

192:
	%193 = load i32, i32* %2
	br label %194

194:
	%195 = phi i32 [ 2, %191 ], [ %193, %192 ]
	store i32 %195, i32* %2
	%196 = load i32*, i32** %1
	%197 = bitcast i32* %196 to i32*
	%198 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 0
	store i8 47, i8* %198
	%199 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 1
	store i8 117, i8* %199
	%200 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 2
	store i8 115, i8* %200
	%201 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 3
	store i8 114, i8* %201
	%202 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 4
	store i8 0, i8* %202
	%203 = getelementptr [5 x i8], [5 x i8]* @.str.40, i32 0, i32 0
	%204 = bitcast i8* %203 to i32*
	%205 = call i32 @rt_streq(i32* %197, i32* %204)
	%206 = icmp ne i32 %205, 0
	%207 = zext i1 %206 to i32
	%208 = icmp ne i32 %207, 0
	br i1 %208, label %209, label %210

209:
	br label %212

210:
	%211 = load i32, i32* %2
	br label %212

212:
	%213 = phi i32 [ 2, %209 ], [ %211, %210 ]
	store i32 %213, i32* %2
	%214 = load i32*, i32** %1
	%215 = bitcast i32* %214 to i32*
	%216 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 0
	store i8 47, i8* %216
	%217 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 1
	store i8 101, i8* %217
	%218 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 2
	store i8 116, i8* %218
	%219 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 3
	store i8 99, i8* %219
	%220 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 4
	store i8 0, i8* %220
	%221 = getelementptr [5 x i8], [5 x i8]* @.str.41, i32 0, i32 0
	%222 = bitcast i8* %221 to i32*
	%223 = call i32 @rt_streq(i32* %215, i32* %222)
	%224 = icmp ne i32 %223, 0
	%225 = zext i1 %224 to i32
	%226 = icmp ne i32 %225, 0
	br i1 %226, label %227, label %228

227:
	br label %230

228:
	%229 = load i32, i32* %2
	br label %230

230:
	%231 = phi i32 [ 2, %227 ], [ %229, %228 ]
	store i32 %231, i32* %2
	%232 = load i32*, i32** %1
	%233 = bitcast i32* %232 to i32*
	%234 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 0
	store i8 47, i8* %234
	%235 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 1
	store i8 118, i8* %235
	%236 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 2
	store i8 97, i8* %236
	%237 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 3
	store i8 114, i8* %237
	%238 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 4
	store i8 0, i8* %238
	%239 = getelementptr [5 x i8], [5 x i8]* @.str.42, i32 0, i32 0
	%240 = bitcast i8* %239 to i32*
	%241 = call i32 @rt_streq(i32* %233, i32* %240)
	%242 = icmp ne i32 %241, 0
	%243 = zext i1 %242 to i32
	%244 = icmp ne i32 %243, 0
	br i1 %244, label %245, label %246

245:
	br label %248

246:
	%247 = load i32, i32* %2
	br label %248

248:
	%249 = phi i32 [ 2, %245 ], [ %247, %246 ]
	store i32 %249, i32* %2
	%250 = load i32*, i32** %1
	%251 = bitcast i32* %250 to i32*
	%252 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 0
	store i8 47, i8* %252
	%253 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 1
	store i8 98, i8* %253
	%254 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 2
	store i8 105, i8* %254
	%255 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 3
	store i8 110, i8* %255
	%256 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 4
	store i8 0, i8* %256
	%257 = getelementptr [5 x i8], [5 x i8]* @.str.43, i32 0, i32 0
	%258 = bitcast i8* %257 to i32*
	%259 = call i32 @rt_streq(i32* %251, i32* %258)
	%260 = icmp ne i32 %259, 0
	%261 = zext i1 %260 to i32
	%262 = icmp ne i32 %261, 0
	br i1 %262, label %263, label %264

263:
	br label %266

264:
	%265 = load i32, i32* %2
	br label %266

266:
	%267 = phi i32 [ 2, %263 ], [ %265, %264 ]
	store i32 %267, i32* %2
	%268 = load i32*, i32** %1
	%269 = bitcast i32* %268 to i32*
	%270 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 0
	store i8 47, i8* %270
	%271 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 1
	store i8 117, i8* %271
	%272 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 2
	store i8 115, i8* %272
	%273 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 3
	store i8 114, i8* %273
	%274 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 4
	store i8 47, i8* %274
	%275 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 5
	store i8 98, i8* %275
	%276 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 6
	store i8 105, i8* %276
	%277 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 7
	store i8 110, i8* %277
	%278 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 8
	store i8 0, i8* %278
	%279 = getelementptr [9 x i8], [9 x i8]* @.str.44, i32 0, i32 0
	%280 = bitcast i8* %279 to i32*
	%281 = call i32 @rt_streq(i32* %269, i32* %280)
	%282 = icmp ne i32 %281, 0
	%283 = zext i1 %282 to i32
	%284 = icmp ne i32 %283, 0
	br i1 %284, label %285, label %286

285:
	br label %288

286:
	%287 = load i32, i32* %2
	br label %288

288:
	%289 = phi i32 [ 2, %285 ], [ %287, %286 ]
	store i32 %289, i32* %2
	%290 = load i32, i32* %2
	ret i32 %290

dead143:
	ret i32 0
}

define i32 @re_get(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = load i32, i32* %2
	%5 = mul i32 %4, 3
	%6 = load i32, i32* %3
	%7 = add i32 %5, %6
	%8 = getelementptr [12288 x i32], [12288 x i32]* @re_prog, i32 0, i32 %7
	%9 = load i32, i32* %8
	ret i32 %9

dead144:
	ret i32 0
}

define i32 @re_set(i32 %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32
	store i32 %0, i32* %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = load i32, i32* %3
	%7 = mul i32 %6, 3
	%8 = load i32, i32* %4
	%9 = add i32 %7, %8
	%10 = getelementptr [12288 x i32], [12288 x i32]* @re_prog, i32 0, i32 %9
	%11 = load i32, i32* %5
	store i32 %11, i32* %10
	ret i32 0

dead145:
	ret i32 0
}

define i32 @re_emit(i32 %0, i32 %1, i32 %2) {
entry:
	%3 = alloca i32
	store i32 %0, i32* %3
	%4 = alloca i32
	store i32 %1, i32* %4
	%5 = alloca i32
	store i32 %2, i32* %5
	%6 = alloca i32
	%7 = load i32, i32* @re_pc
	store i32 %7, i32* %6
	%8 = load i32, i32* %6
	%9 = icmp sge i32 %8, u0x1000
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %12, label %13

12:
	store i32 4, i32* @re_bad
	ret i32 0

13:
	%14 = load i32, i32* %6
	%15 = load i32, i32* %3
	%16 = call i32 @re_set(i32 %14, i32 0, i32 %15)
	%17 = load i32, i32* %6
	%18 = load i32, i32* %4
	%19 = call i32 @re_set(i32 %17, i32 1, i32 %18)
	%20 = load i32, i32* %6
	%21 = load i32, i32* %5
	%22 = call i32 @re_set(i32 %20, i32 2, i32 %21)
	%23 = load i32, i32* %6
	%24 = add i32 %23, 1
	store i32 %24, i32* @re_pc
	%25 = load i32, i32* %6
	ret i32 %25

dead146:
	br label %13

dead147:
	ret i32 0
}

define i32 @re_mark() {
entry:
	%0 = alloca i32
	%1 = load i32, i32* @re_nmark
	store i32 %1, i32* %0
	%2 = load i32, i32* %0
	%3 = add i32 %2, 1
	store i32 %3, i32* @re_nmark
	%4 = load i32, i32* %0
	%5 = srem i32 %4, 64
	%6 = add i32 128, %5
	ret i32 %6

dead148:
	ret i32 0
}

define i32 @re_at(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32*
	%3 = alloca i32
	%4 = alloca i32
	%5 = alloca i32
	%6 = load i32*, i32** @re_pat
	%7 = bitcast i32* %6 to i32*
	store i32* %7, i32** %2
	%8 = load i32, i32* @re_pos
	store i32 %8, i32* %3
	store i32 0, i32* %4
	store i32 0, i32* %5
	br label %9

9:
	%10 = icmp ne i32 1, 0
	br i1 %10, label %11, label %24

11:
	%12 = load i32, i32* %3
	%13 = load i32*, i32** %2
	%14 = getelementptr i8, i32* %13, i32 %12
	%15 = load i8, i8* %14
	%16 = sext i8 %15 to i32
	%17 = and i32 %16, 255
	store i32 %17, i32* %5
	%18 = load i32, i32* %5
	%19 = icmp ne i32 %18, 0
	%20 = zext i1 %19 to i32
	%21 = icmp ne i32 %20, 0
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	br i1 %23, label %26, label %33

24:
	%25 = load i32, i32* %5
	ret i32 %25

26:
	%27 = load i32, i32* %4
	%28 = load i32, i32* %1
	%29 = icmp slt i32 %27, %28
	%30 = zext i1 %29 to i32
	%31 = icmp ne i32 %30, 0
	%32 = zext i1 %31 to i32
	br label %33

33:
	%34 = phi i32 [ %22, %11 ], [ %32, %26 ]
	%35 = icmp eq i32 %34, 0
	%36 = zext i1 %35 to i32
	%37 = icmp ne i32 %36, 0
	br i1 %37, label %38, label %39

38:
	br label %24

39:
	%40 = load i32, i32* %3
	%41 = add i32 %40, 1
	store i32 %41, i32* %3
	%42 = load i32, i32* %4
	%43 = add i32 %42, 1
	store i32 %43, i32* %4
	br label %9

dead149:
	br label %39

dead150:
	ret i32 0
}

define i32 @re_num() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	store i32 0, i32* %0
	br label %2

2:
	%3 = icmp ne i32 1, 0
	br i1 %3, label %4, label %12

4:
	%5 = call i32 @re_at(i32 0)
	store i32 %5, i32* %1
	%6 = load i32, i32* %1
	%7 = icmp sge i32 %6, 48
	%8 = zext i1 %7 to i32
	%9 = icmp ne i32 %8, 0
	%10 = zext i1 %9 to i32
	%11 = icmp ne i32 %10, 0
	br i1 %11, label %14, label %20

12:
	%13 = load i32, i32* %0
	ret i32 %13

14:
	%15 = load i32, i32* %1
	%16 = icmp sle i32 %15, 57
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	%19 = zext i1 %18 to i32
	br label %20

20:
	%21 = phi i32 [ %10, %4 ], [ %19, %14 ]
	%22 = icmp eq i32 %21, 0
	%23 = zext i1 %22 to i32
	%24 = icmp ne i32 %23, 0
	br i1 %24, label %25, label %26

25:
	br label %12

26:
	%27 = load i32, i32* %0
	%28 = icmp slt i32 %27, 256
	%29 = zext i1 %28 to i32
	%30 = icmp ne i32 %29, 0
	br i1 %30, label %31, label %37

dead151:
	br label %26

31:
	%32 = load i32, i32* %0
	%33 = mul i32 %32, 10
	%34 = load i32, i32* %1
	%35 = sub i32 %34, 48
	%36 = add i32 %33, %35
	br label %38

37:
	br label %38

38:
	%39 = phi i32 [ %36, %31 ], [ 256, %37 ]
	store i32 %39, i32* %0
	%40 = load i32, i32* @re_pos
	%41 = add i32 %40, 1
	store i32 %41, i32* @re_pos
	br label %2

dead152:
	ret i32 0
}

define i32 @re_isq() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	%4 = call i32 @re_at(i32 0)
	store i32 %4, i32* %0
	%5 = call i32 @re_at(i32 1)
	store i32 %5, i32* %1
	%6 = load i32, i32* %0
	%7 = icmp eq i32 %6, 42
	%8 = zext i1 %7 to i32
	%9 = load i32, i32* %0
	%10 = icmp eq i32 %9, 43
	%11 = zext i1 %10 to i32
	%12 = or i32 %8, %11
	%13 = load i32, i32* %0
	%14 = icmp eq i32 %13, 63
	%15 = zext i1 %14 to i32
	%16 = or i32 %12, %15
	store i32 %16, i32* %2
	%17 = load i32, i32* %0
	%18 = icmp eq i32 %17, 123
	%19 = zext i1 %18 to i32
	%20 = load i32, i32* %1
	%21 = icmp sge i32 %20, 48
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %32

26:
	%27 = load i32, i32* %1
	%28 = icmp sle i32 %27, 57
	%29 = zext i1 %28 to i32
	%30 = icmp ne i32 %29, 0
	%31 = zext i1 %30 to i32
	br label %32

32:
	%33 = phi i32 [ %24, %entry ], [ %31, %26 ]
	%34 = and i32 %19, %33
	store i32 %34, i32* %3
	%35 = load i32, i32* %2
	%36 = load i32, i32* %3
	%37 = or i32 %35, %36
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %39, label %40

39:
	br label %41

40:
	br label %41

41:
	%42 = phi i32 [ 1, %39 ], [ 0, %40 ]
	ret i32 %42

dead153:
	ret i32 0
}

define i32 @re_fold1(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = load i32, i32* %1
	%4 = icmp sge i32 %3, 65
	%5 = zext i1 %4 to i32
	%6 = icmp ne i32 %5, 0
	%7 = zext i1 %6 to i32
	%8 = icmp ne i32 %7, 0
	br i1 %8, label %9, label %15

9:
	%10 = load i32, i32* %1
	%11 = icmp sle i32 %10, 90
	%12 = zext i1 %11 to i32
	%13 = icmp ne i32 %12, 0
	%14 = zext i1 %13 to i32
	br label %15

15:
	%16 = phi i32 [ %7, %entry ], [ %14, %9 ]
	store i32 %16, i32* %2
	%17 = load i32, i32* @re_fold
	%18 = icmp ne i32 %17, 0
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	%21 = zext i1 %20 to i32
	%22 = icmp ne i32 %21, 0
	br i1 %22, label %23, label %27

23:
	%24 = load i32, i32* %2
	%25 = icmp ne i32 %24, 0
	%26 = zext i1 %25 to i32
	br label %27

27:
	%28 = phi i32 [ %21, %15 ], [ %26, %23 ]
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %30, label %33

30:
	%31 = load i32, i32* %1
	%32 = add i32 %31, 32
	br label %35

33:
	%34 = load i32, i32* %1
	br label %35

35:
	%36 = phi i32 [ %32, %30 ], [ %34, %33 ]
	ret i32 %36

dead154:
	ret i32 0
}

define i32 @re_bit1(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = load i32, i32* %2
	%6 = mul i32 %5, 32
	%7 = load i32, i32* %3
	%8 = sdiv i32 %7, 8
	%9 = add i32 %6, %8
	store i32 %9, i32* %4
	%10 = load i32, i32* %4
	%11 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %10
	%12 = load i32, i32* %4
	%13 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %12
	%14 = load i8, i8* %13
	%15 = zext i8 %14 to i32
	%16 = load i32, i32* %3
	%17 = srem i32 %16, 8
	%18 = shl i32 1, %17
	%19 = and i32 %18, 255
	%20 = or i32 %15, %19
	%21 = and i32 %20, 255
	%22 = and i32 %21, 255
	%23 = trunc i32 %22 to i8
	store i8 %23, i8* %11
	ret i32 0

dead155:
	ret i32 0
}

define i32 @re_setbit(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = load i32, i32* %2
	%7 = load i32, i32* %3
	%8 = call i32 @re_bit1(i32 %6, i32 %7)
	%9 = load i32, i32* %3
	%10 = icmp sge i32 %9, 65
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	%13 = zext i1 %12 to i32
	%14 = icmp ne i32 %13, 0
	br i1 %14, label %15, label %21

15:
	%16 = load i32, i32* %3
	%17 = icmp sle i32 %16, 90
	%18 = zext i1 %17 to i32
	%19 = icmp ne i32 %18, 0
	%20 = zext i1 %19 to i32
	br label %21

21:
	%22 = phi i32 [ %13, %entry ], [ %20, %15 ]
	store i32 %22, i32* %4
	%23 = load i32, i32* %3
	%24 = icmp sge i32 %23, 97
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	%27 = zext i1 %26 to i32
	%28 = icmp ne i32 %27, 0
	br i1 %28, label %29, label %35

29:
	%30 = load i32, i32* %3
	%31 = icmp sle i32 %30, 122
	%32 = zext i1 %31 to i32
	%33 = icmp ne i32 %32, 0
	%34 = zext i1 %33 to i32
	br label %35

35:
	%36 = phi i32 [ %27, %21 ], [ %34, %29 ]
	store i32 %36, i32* %5
	%37 = load i32, i32* @re_fold
	%38 = icmp ne i32 %37, 0
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	%41 = zext i1 %40 to i32
	%42 = icmp ne i32 %41, 0
	br i1 %42, label %43, label %49

43:
	%44 = load i32, i32* %4
	%45 = load i32, i32* %5
	%46 = or i32 %44, %45
	%47 = icmp ne i32 %46, 0
	%48 = zext i1 %47 to i32
	br label %49

49:
	%50 = phi i32 [ %41, %35 ], [ %48, %43 ]
	%51 = icmp ne i32 %50, 0
	br i1 %51, label %52, label %56

52:
	%53 = load i32, i32* %2
	%54 = load i32, i32* %4
	%55 = icmp ne i32 %54, 0
	br i1 %55, label %57, label %60

56:
	ret i32 0

57:
	%58 = load i32, i32* %3
	%59 = add i32 %58, 32
	br label %63

60:
	%61 = load i32, i32* %3
	%62 = sub i32 %61, 32
	br label %63

63:
	%64 = phi i32 [ %59, %57 ], [ %62, %60 ]
	%65 = call i32 @re_bit1(i32 %53, i32 %64)
	br label %56

dead156:
	ret i32 0
}

define i32 @re_clrcls(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = load i32, i32* %1
	%5 = mul i32 %4, 32
	store i32 %5, i32* %3
	store i32 0, i32* %2
	br label %6

6:
	%7 = load i32, i32* %2
	%8 = icmp slt i32 %7, 32
	%9 = zext i1 %8 to i32
	%10 = icmp ne i32 %9, 0
	br i1 %10, label %11, label %22

11:
	%12 = load i32, i32* %3
	%13 = load i32, i32* %2
	%14 = add i32 %12, %13
	%15 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %14
	%16 = and i32 0, 255
	%17 = and i32 %16, 255
	%18 = trunc i32 %17 to i8
	store i8 %18, i8* %15
	br label %19

19:
	%20 = load i32, i32* %2
	%21 = add i32 %20, 1
	store i32 %21, i32* %2
	br label %6

22:
	ret i32 0

dead157:
	ret i32 0
}

define i32 @re_negcls(i32 %0) {
entry:
	%1 = alloca i32
	store i32 %0, i32* %1
	%2 = alloca i32
	%3 = alloca i32
	%4 = load i32, i32* %1
	%5 = mul i32 %4, 32
	store i32 %5, i32* %3
	store i32 0, i32* %2
	br label %6

6:
	%7 = load i32, i32* %2
	%8 = icmp slt i32 %7, 32
	%9 = zext i1 %8 to i32
	%10 = icmp ne i32 %9, 0
	br i1 %10, label %11, label %29

11:
	%12 = load i32, i32* %3
	%13 = load i32, i32* %2
	%14 = add i32 %12, %13
	%15 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %14
	%16 = load i32, i32* %3
	%17 = load i32, i32* %2
	%18 = add i32 %16, %17
	%19 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %18
	%20 = load i8, i8* %19
	%21 = zext i8 %20 to i32
	%22 = xor i32 %21, 255
	%23 = and i32 %22, 255
	%24 = and i32 %23, 255
	%25 = trunc i32 %24 to i8
	store i8 %25, i8* %15
	br label %26

26:
	%27 = load i32, i32* %2
	%28 = add i32 %27, 1
	store i32 %28, i32* %2
	br label %6

29:
	ret i32 0

dead158:
	ret i32 0
}

define i32 @re_clsname() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32*
	%3 = alloca i32
	store i32 0, i32* %0
	br label %4

4:
	%5 = icmp ne i32 1, 0
	br i1 %5, label %6, label %13

6:
	%7 = load i32, i32* %0
	%8 = call i32 @re_at(i32 %7)
	store i32 %8, i32* %1
	%9 = load i32, i32* %1
	%10 = icmp eq i32 %9, 0
	%11 = zext i1 %10 to i32
	%12 = icmp ne i32 %11, 0
	br i1 %12, label %18, label %19

13:
	%14 = load i32, i32* %0
	%15 = add i32 %14, 1
	%16 = call i32* @rt_bump(i32 %15)
	%17 = bitcast i32* %16 to i32*
	store i32* %17, i32** %2
	store i32 0, i32* %1
	br label %41

18:
	ret i32 0

19:
	%20 = load i32, i32* %1
	%21 = icmp eq i32 %20, 58
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %26, label %34

dead160:
	br label %19

26:
	%27 = load i32, i32* %0
	%28 = add i32 %27, 1
	%29 = call i32 @re_at(i32 %28)
	%30 = icmp eq i32 %29, 93
	%31 = zext i1 %30 to i32
	%32 = icmp ne i32 %31, 0
	%33 = zext i1 %32 to i32
	br label %34

34:
	%35 = phi i32 [ %24, %19 ], [ %33, %26 ]
	%36 = icmp ne i32 %35, 0
	br i1 %36, label %37, label %38

37:
	br label %13

38:
	%39 = load i32, i32* %0
	%40 = add i32 %39, 1
	store i32 %40, i32* %0
	br label %4

dead161:
	br label %38

41:
	%42 = load i32, i32* %1
	%43 = load i32, i32* %0
	%44 = icmp slt i32 %42, %43
	%45 = zext i1 %44 to i32
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %62

47:
	%48 = load i32, i32* %1
	%49 = load i32*, i32** %2
	%50 = getelementptr i8, i32* %49, i32 %48
	%51 = load i32, i32* %1
	%52 = call i32 @re_at(i32 %51)
	%53 = shl i32 %52, 24
	%54 = ashr i32 %53, 24
	%55 = shl i32 %54, 24
	%56 = ashr i32 %55, 24
	%57 = shl i32 %56, 24
	%58 = ashr i32 %57, 24
	%59 = trunc i32 %58 to i8
	store i8 %59, i8* %50
	%60 = load i32, i32* %1
	%61 = add i32 %60, 1
	store i32 %61, i32* %1
	br label %41

62:
	%63 = load i32, i32* %0
	%64 = load i32*, i32** %2
	%65 = getelementptr i8, i32* %64, i32 %63
	%66 = shl i32 0, 24
	%67 = ashr i32 %66, 24
	%68 = shl i32 %67, 24
	%69 = ashr i32 %68, 24
	%70 = trunc i32 %69 to i8
	store i8 %70, i8* %65
	%71 = load i32*, i32** %2
	%72 = bitcast i32* %71 to i32*
	%73 = call i32 @rt_clsid(i32* %72)
	store i32 %73, i32* %3
	%74 = load i32, i32* %3
	%75 = icmp eq i32 %74, 0
	%76 = zext i1 %75 to i32
	%77 = icmp ne i32 %76, 0
	br i1 %77, label %78, label %79

78:
	ret i32 0

79:
	%80 = load i32, i32* @re_pos
	%81 = load i32, i32* %0
	%82 = add i32 %81, 2
	%83 = add i32 %80, %82
	store i32 %83, i32* @re_pos
	%84 = load i32, i32* %3
	ret i32 %84

dead162:
	br label %79

dead163:
	ret i32 0
}

define i32 @re_bracket() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = load i32, i32* @re_ncls
	store i32 %14, i32* %0
	%15 = load i32, i32* %0
	%16 = icmp sge i32 %15, 64
	%17 = zext i1 %16 to i32
	%18 = icmp ne i32 %17, 0
	br i1 %18, label %19, label %20

19:
	store i32 4, i32* @re_bad
	ret i32 0

20:
	%21 = load i32, i32* %0
	%22 = add i32 %21, 1
	store i32 %22, i32* @re_ncls
	%23 = load i32, i32* %0
	%24 = call i32 @re_clrcls(i32 %23)
	%25 = load i32, i32* @re_pos
	%26 = add i32 %25, 1
	store i32 %26, i32* @re_pos
	%27 = call i32 @re_at(i32 0)
	store i32 %27, i32* %1
	%28 = load i32, i32* %1
	%29 = icmp eq i32 %28, 94
	%30 = zext i1 %29 to i32
	store i32 %30, i32* %2
	%31 = load i32, i32* %2
	%32 = icmp ne i32 %31, 0
	br i1 %32, label %33, label %34

dead164:
	br label %20

33:
	br label %35

34:
	br label %35

35:
	%36 = phi i32 [ 1, %33 ], [ 0, %34 ]
	store i32 %36, i32* %3
	%37 = load i32, i32* @re_pos
	%38 = load i32, i32* %2
	%39 = icmp ne i32 %38, 0
	br i1 %39, label %40, label %41

40:
	br label %42

41:
	br label %42

42:
	%43 = phi i32 [ 1, %40 ], [ 0, %41 ]
	%44 = add i32 %37, %43
	store i32 %44, i32* @re_pos
	store i32 1, i32* %4
	br label %45

45:
	%46 = icmp ne i32 1, 0
	br i1 %46, label %47, label %53

47:
	%48 = call i32 @re_at(i32 0)
	store i32 %48, i32* %5
	%49 = load i32, i32* %5
	%50 = icmp eq i32 %49, 0
	%51 = zext i1 %50 to i32
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %60, label %61

53:
	%54 = load i32, i32* @re_pos
	%55 = add i32 %54, 1
	store i32 %55, i32* @re_pos
	%56 = load i32, i32* %3
	%57 = icmp ne i32 %56, 0
	%58 = zext i1 %57 to i32
	%59 = icmp ne i32 %58, 0
	br i1 %59, label %235, label %238

60:
	store i32 2, i32* @re_bad
	ret i32 0

61:
	%62 = load i32, i32* %5
	%63 = icmp eq i32 %62, 93
	%64 = zext i1 %63 to i32
	%65 = icmp ne i32 %64, 0
	%66 = zext i1 %65 to i32
	%67 = icmp ne i32 %66, 0
	br i1 %67, label %68, label %74

dead165:
	br label %61

68:
	%69 = load i32, i32* %4
	%70 = icmp eq i32 %69, 0
	%71 = zext i1 %70 to i32
	%72 = icmp ne i32 %71, 0
	%73 = zext i1 %72 to i32
	br label %74

74:
	%75 = phi i32 [ %66, %61 ], [ %73, %68 ]
	%76 = icmp ne i32 %75, 0
	br i1 %76, label %77, label %78

77:
	br label %53

78:
	store i32 0, i32* %4
	%79 = load i32, i32* %5
	%80 = icmp eq i32 %79, 91
	%81 = zext i1 %80 to i32
	%82 = icmp ne i32 %81, 0
	%83 = zext i1 %82 to i32
	%84 = icmp ne i32 %83, 0
	br i1 %84, label %85, label %91

dead166:
	br label %78

85:
	%86 = call i32 @re_at(i32 1)
	%87 = icmp eq i32 %86, 58
	%88 = zext i1 %87 to i32
	%89 = icmp ne i32 %88, 0
	%90 = zext i1 %89 to i32
	br label %91

91:
	%92 = phi i32 [ %83, %78 ], [ %90, %85 ]
	%93 = icmp ne i32 %92, 0
	br i1 %93, label %94, label %102

94:
	%95 = load i32, i32* @re_pos
	%96 = add i32 %95, 2
	store i32 %96, i32* @re_pos
	%97 = call i32 @re_clsname()
	store i32 %97, i32* %6
	%98 = load i32, i32* %6
	%99 = icmp eq i32 %98, 0
	%100 = zext i1 %99 to i32
	%101 = icmp ne i32 %100, 0
	br i1 %101, label %114, label %115

102:
	%103 = load i32, i32* @re_pos
	%104 = add i32 %103, 1
	store i32 %104, i32* @re_pos
	%105 = call i32 @re_at(i32 0)
	store i32 %105, i32* %11
	%106 = call i32 @re_at(i32 1)
	store i32 %106, i32* %12
	%107 = call i32 @re_at(i32 2)
	store i32 %107, i32* %13
	%108 = load i32, i32* %11
	%109 = icmp eq i32 %108, 45
	%110 = zext i1 %109 to i32
	%111 = icmp ne i32 %110, 0
	%112 = zext i1 %111 to i32
	%113 = icmp ne i32 %112, 0
	br i1 %113, label %167, label %174

114:
	store i32 9, i32* @re_bad
	ret i32 0

115:
	store i32 0, i32* %7
	br label %116

dead167:
	br label %115

116:
	%117 = load i32, i32* %7
	%118 = icmp slt i32 %117, 256
	%119 = zext i1 %118 to i32
	%120 = icmp ne i32 %119, 0
	br i1 %120, label %121, label %131

121:
	%122 = load i32, i32* %6
	%123 = load i32, i32* %7
	%124 = call i32 @re_ctype(i32 %122, i32 %123)
	%125 = icmp ne i32 %124, 0
	%126 = zext i1 %125 to i32
	%127 = icmp ne i32 %126, 0
	br i1 %127, label %140, label %144

128:
	%129 = load i32, i32* %7
	%130 = add i32 %129, 1
	store i32 %130, i32* %7
	br label %116

131:
	%132 = call i32 @re_at(i32 0)
	store i32 %132, i32* %9
	%133 = call i32 @re_at(i32 1)
	store i32 %133, i32* %10
	%134 = load i32, i32* %9
	%135 = icmp eq i32 %134, 45
	%136 = zext i1 %135 to i32
	%137 = icmp ne i32 %136, 0
	%138 = zext i1 %137 to i32
	%139 = icmp ne i32 %138, 0
	br i1 %139, label %145, label %152

140:
	%141 = load i32, i32* %0
	%142 = load i32, i32* %7
	%143 = call i32 @re_setbit(i32 %141, i32 %142)
	br label %144

144:
	br label %128

145:
	%146 = load i32, i32* %10
	%147 = icmp ne i32 %146, 93
	%148 = zext i1 %147 to i32
	%149 = icmp ne i32 %148, 0
	%150 = zext i1 %149 to i32
	%151 = icmp ne i32 %150, 0
	br i1 %151, label %155, label %161

152:
	%153 = phi i32 [ %138, %131 ], [ %164, %161 ]
	%154 = icmp ne i32 %153, 0
	br i1 %154, label %165, label %166

155:
	%156 = load i32, i32* %10
	%157 = icmp ne i32 %156, 0
	%158 = zext i1 %157 to i32
	%159 = icmp ne i32 %158, 0
	%160 = zext i1 %159 to i32
	br label %161

161:
	%162 = phi i32 [ %150, %145 ], [ %160, %155 ]
	%163 = icmp ne i32 %162, 0
	%164 = zext i1 %163 to i32
	br label %152

165:
	store i32 7, i32* @re_bad
	ret i32 0

166:
	br label %45

dead168:
	br label %166

dead169:
	br label %102

167:
	%168 = load i32, i32* %12
	%169 = icmp ne i32 %168, 93
	%170 = zext i1 %169 to i32
	%171 = icmp ne i32 %170, 0
	%172 = zext i1 %171 to i32
	%173 = icmp ne i32 %172, 0
	br i1 %173, label %177, label %183

174:
	%175 = phi i32 [ %112, %102 ], [ %186, %183 ]
	%176 = icmp ne i32 %175, 0
	br i1 %176, label %187, label %194

177:
	%178 = load i32, i32* %12
	%179 = icmp ne i32 %178, 0
	%180 = zext i1 %179 to i32
	%181 = icmp ne i32 %180, 0
	%182 = zext i1 %181 to i32
	br label %183

183:
	%184 = phi i32 [ %172, %167 ], [ %182, %177 ]
	%185 = icmp ne i32 %184, 0
	%186 = zext i1 %185 to i32
	br label %174

187:
	%188 = load i32, i32* %12
	%189 = icmp eq i32 %188, 91
	%190 = zext i1 %189 to i32
	%191 = icmp ne i32 %190, 0
	%192 = zext i1 %191 to i32
	%193 = icmp ne i32 %192, 0
	br i1 %193, label %198, label %204

194:
	%195 = load i32, i32* %0
	%196 = load i32, i32* %5
	%197 = call i32 @re_setbit(i32 %195, i32 %196)
	br label %45

198:
	%199 = load i32, i32* %13
	%200 = icmp eq i32 %199, 58
	%201 = zext i1 %200 to i32
	%202 = icmp ne i32 %201, 0
	%203 = zext i1 %202 to i32
	br label %204

204:
	%205 = phi i32 [ %192, %187 ], [ %203, %198 ]
	%206 = icmp ne i32 %205, 0
	br i1 %206, label %207, label %208

207:
	store i32 7, i32* @re_bad
	ret i32 0

208:
	%209 = load i32, i32* @re_pos
	%210 = add i32 %209, 2
	store i32 %210, i32* @re_pos
	%211 = load i32, i32* %5
	%212 = load i32, i32* %12
	%213 = icmp sgt i32 %211, %212
	%214 = zext i1 %213 to i32
	%215 = icmp ne i32 %214, 0
	br i1 %215, label %216, label %217

dead170:
	br label %208

216:
	store i32 7, i32* @re_bad
	ret i32 0

217:
	%218 = load i32, i32* %12
	%219 = add i32 %218, 1
	store i32 %219, i32* %8
	%220 = load i32, i32* %5
	store i32 %220, i32* %7
	br label %221

dead171:
	br label %217

221:
	%222 = load i32, i32* %7
	%223 = load i32, i32* %8
	%224 = icmp slt i32 %222, %223
	%225 = zext i1 %224 to i32
	%226 = icmp ne i32 %225, 0
	br i1 %226, label %227, label %234

227:
	%228 = load i32, i32* %0
	%229 = load i32, i32* %7
	%230 = call i32 @re_setbit(i32 %228, i32 %229)
	br label %231

231:
	%232 = load i32, i32* %7
	%233 = add i32 %232, 1
	store i32 %233, i32* %7
	br label %221

234:
	br label %45

dead172:
	br label %194

235:
	%236 = load i32, i32* %0
	%237 = call i32 @re_negcls(i32 %236)
	br label %238

238:
	%239 = load i32, i32* %0
	%240 = call i32 @re_emit(i32 3, i32 %239, i32 0)
	ret i32 0

dead173:
	ret i32 0
}

define i32 @re_copy(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = load i32, i32* @re_pc
	%15 = load i32, i32* %2
	%16 = sub i32 %14, %15
	store i32 %16, i32* %4
	%17 = load i32, i32* %2
	store i32 %17, i32* %5
	br label %18

18:
	%19 = load i32, i32* %5
	%20 = load i32, i32* %3
	%21 = icmp slt i32 %19, %20
	%22 = zext i1 %21 to i32
	%23 = icmp ne i32 %22, 0
	%24 = zext i1 %23 to i32
	%25 = icmp ne i32 %24, 0
	br i1 %25, label %66, label %72

26:
	%27 = load i32, i32* %5
	%28 = call i32 @re_get(i32 %27, i32 0)
	store i32 %28, i32* %6
	%29 = load i32, i32* %5
	%30 = call i32 @re_get(i32 %29, i32 1)
	store i32 %30, i32* %7
	%31 = load i32, i32* %5
	%32 = call i32 @re_get(i32 %31, i32 2)
	store i32 %32, i32* %8
	%33 = load i32, i32* %6
	%34 = icmp eq i32 %33, 5
	%35 = zext i1 %34 to i32
	%36 = load i32, i32* %6
	%37 = icmp eq i32 %36, 4
	%38 = zext i1 %37 to i32
	%39 = or i32 %35, %38
	store i32 %39, i32* %9
	%40 = load i32, i32* %7
	%41 = load i32, i32* %2
	%42 = icmp sge i32 %40, %41
	%43 = zext i1 %42 to i32
	%44 = load i32, i32* %7
	%45 = load i32, i32* %3
	%46 = icmp sle i32 %44, %45
	%47 = zext i1 %46 to i32
	%48 = and i32 %43, %47
	store i32 %48, i32* %10
	%49 = load i32, i32* %8
	%50 = load i32, i32* %2
	%51 = icmp sge i32 %49, %50
	%52 = zext i1 %51 to i32
	%53 = load i32, i32* %8
	%54 = load i32, i32* %3
	%55 = icmp sle i32 %53, %54
	%56 = zext i1 %55 to i32
	%57 = and i32 %52, %56
	store i32 %57, i32* %11
	%58 = load i32, i32* %9
	%59 = load i32, i32* %10
	%60 = and i32 %58, %59
	%61 = icmp ne i32 %60, 0
	br i1 %61, label %75, label %79

62:
	%63 = load i32, i32* %5
	%64 = add i32 %63, 1
	store i32 %64, i32* %5
	br label %18

65:
	ret i32 0

66:
	%67 = load i32, i32* @re_bad
	%68 = icmp eq i32 %67, 0
	%69 = zext i1 %68 to i32
	%70 = icmp ne i32 %69, 0
	%71 = zext i1 %70 to i32
	br label %72

72:
	%73 = phi i32 [ %24, %18 ], [ %71, %66 ]
	%74 = icmp ne i32 %73, 0
	br i1 %74, label %26, label %65

75:
	%76 = load i32, i32* %7
	%77 = load i32, i32* %4
	%78 = add i32 %76, %77
	br label %81

79:
	%80 = load i32, i32* %7
	br label %81

81:
	%82 = phi i32 [ %78, %75 ], [ %80, %79 ]
	store i32 %82, i32* %12
	%83 = load i32, i32* %6
	%84 = icmp eq i32 %83, 4
	%85 = zext i1 %84 to i32
	%86 = load i32, i32* %11
	%87 = and i32 %85, %86
	%88 = icmp ne i32 %87, 0
	br i1 %88, label %89, label %93

89:
	%90 = load i32, i32* %8
	%91 = load i32, i32* %4
	%92 = add i32 %90, %91
	br label %95

93:
	%94 = load i32, i32* %8
	br label %95

95:
	%96 = phi i32 [ %92, %89 ], [ %94, %93 ]
	store i32 %96, i32* %13
	%97 = load i32, i32* %6
	%98 = load i32, i32* %12
	%99 = load i32, i32* %13
	%100 = call i32 @re_emit(i32 %97, i32 %98, i32 %99)
	br label %62

dead174:
	ret i32 0
}

define i32 @re_atom() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	%4 = call i32 @re_at(i32 0)
	store i32 %4, i32* %0
	%5 = load i32, i32* %0
	%6 = icmp eq i32 %5, 40
	%7 = zext i1 %6 to i32
	%8 = icmp ne i32 %7, 0
	br i1 %8, label %9, label %22

9:
	%10 = load i32, i32* @re_pos
	%11 = add i32 %10, 1
	store i32 %11, i32* @re_pos
	%12 = load i32, i32* @re_ng
	%13 = add i32 %12, 1
	store i32 %13, i32* %1
	%14 = load i32, i32* %1
	store i32 %14, i32* @re_ng
	%15 = load i32, i32* %1
	%16 = mul i32 %15, 2
	%17 = add i32 %16, 1
	%18 = icmp slt i32 %17, 128
	%19 = zext i1 %18 to i32
	store i32 %19, i32* %2
	%20 = load i32, i32* %2
	%21 = icmp ne i32 %20, 0
	br i1 %21, label %27, label %31

22:
	%23 = load i32, i32* %0
	%24 = icmp eq i32 %23, 46
	%25 = zext i1 %24 to i32
	%26 = icmp ne i32 %25, 0
	br i1 %26, label %59, label %63

27:
	%28 = load i32, i32* %1
	%29 = mul i32 %28, 2
	%30 = call i32 @re_emit(i32 6, i32 %29, i32 0)
	br label %31

31:
	%32 = load i32, i32* @re_depth
	%33 = add i32 %32, 1
	store i32 %33, i32* @re_depth
	%34 = call i32 @re_alt()
	%35 = load i32, i32* @re_depth
	%36 = sub i32 %35, 1
	store i32 %36, i32* @re_depth
	%37 = load i32, i32* @re_bad
	%38 = icmp ne i32 %37, 0
	%39 = zext i1 %38 to i32
	%40 = icmp ne i32 %39, 0
	br i1 %40, label %41, label %42

41:
	ret i32 0

42:
	%43 = call i32 @re_at(i32 0)
	%44 = icmp ne i32 %43, 41
	%45 = zext i1 %44 to i32
	%46 = icmp ne i32 %45, 0
	br i1 %46, label %47, label %48

dead175:
	br label %42

47:
	store i32 1, i32* @re_bad
	ret i32 0

48:
	%49 = load i32, i32* @re_pos
	%50 = add i32 %49, 1
	store i32 %50, i32* @re_pos
	%51 = load i32, i32* %2
	%52 = icmp ne i32 %51, 0
	br i1 %52, label %53, label %58

dead176:
	br label %48

53:
	%54 = load i32, i32* %1
	%55 = mul i32 %54, 2
	%56 = add i32 %55, 1
	%57 = call i32 @re_emit(i32 6, i32 %56, i32 0)
	br label %58

58:
	ret i32 0

dead177:
	br label %22

59:
	%60 = load i32, i32* @re_pos
	%61 = add i32 %60, 1
	store i32 %61, i32* @re_pos
	%62 = call i32 @re_emit(i32 2, i32 0, i32 0)
	ret i32 0

63:
	%64 = load i32, i32* %0
	%65 = icmp eq i32 %64, 94
	%66 = zext i1 %65 to i32
	%67 = icmp ne i32 %66, 0
	br i1 %67, label %68, label %72

dead178:
	br label %63

68:
	%69 = load i32, i32* @re_pos
	%70 = add i32 %69, 1
	store i32 %70, i32* @re_pos
	%71 = call i32 @re_emit(i32 8, i32 0, i32 0)
	ret i32 0

72:
	%73 = load i32, i32* %0
	%74 = icmp eq i32 %73, 36
	%75 = zext i1 %74 to i32
	%76 = icmp ne i32 %75, 0
	br i1 %76, label %77, label %81

dead179:
	br label %72

77:
	%78 = load i32, i32* @re_pos
	%79 = add i32 %78, 1
	store i32 %79, i32* @re_pos
	%80 = call i32 @re_emit(i32 9, i32 0, i32 0)
	ret i32 0

81:
	%82 = load i32, i32* %0
	%83 = icmp eq i32 %82, 91
	%84 = zext i1 %83 to i32
	%85 = icmp ne i32 %84, 0
	br i1 %85, label %86, label %88

dead180:
	br label %81

86:
	%87 = call i32 @re_bracket()
	ret i32 0

88:
	%89 = load i32, i32* %0
	%90 = icmp eq i32 %89, 92
	%91 = zext i1 %90 to i32
	%92 = icmp ne i32 %91, 0
	br i1 %92, label %93, label %99

dead181:
	br label %88

93:
	%94 = call i32 @re_at(i32 1)
	store i32 %94, i32* %3
	%95 = load i32, i32* %3
	%96 = icmp eq i32 %95, 0
	%97 = zext i1 %96 to i32
	%98 = icmp ne i32 %97, 0
	br i1 %98, label %106, label %107

99:
	%100 = load i32, i32* %0
	%101 = icmp eq i32 %100, 42
	%102 = zext i1 %101 to i32
	%103 = icmp ne i32 %102, 0
	%104 = zext i1 %103 to i32
	%105 = icmp ne i32 %104, 0
	br i1 %105, label %119, label %113

106:
	store i32 6, i32* @re_bad
	ret i32 0

107:
	%108 = load i32, i32* @re_pos
	%109 = add i32 %108, 2
	store i32 %109, i32* @re_pos
	%110 = load i32, i32* %3
	%111 = call i32 @re_fold1(i32 %110)
	%112 = call i32 @re_emit(i32 1, i32 %111, i32 0)
	ret i32 0

dead182:
	br label %107

dead183:
	br label %99

113:
	%114 = load i32, i32* %0
	%115 = icmp eq i32 %114, 43
	%116 = zext i1 %115 to i32
	%117 = icmp ne i32 %116, 0
	%118 = zext i1 %117 to i32
	br label %119

119:
	%120 = phi i32 [ %104, %99 ], [ %118, %113 ]
	%121 = icmp ne i32 %120, 0
	br i1 %121, label %128, label %122

122:
	%123 = load i32, i32* %0
	%124 = icmp eq i32 %123, 63
	%125 = zext i1 %124 to i32
	%126 = icmp ne i32 %125, 0
	%127 = zext i1 %126 to i32
	br label %128

128:
	%129 = phi i32 [ %120, %119 ], [ %127, %122 ]
	%130 = icmp ne i32 %129, 0
	br i1 %130, label %131, label %132

131:
	store i32 5, i32* @re_bad
	ret i32 0

132:
	%133 = load i32, i32* @re_pos
	%134 = add i32 %133, 1
	store i32 %134, i32* @re_pos
	%135 = load i32, i32* %0
	%136 = call i32 @re_fold1(i32 %135)
	%137 = call i32 @re_emit(i32 1, i32 %136, i32 0)
	ret i32 0

dead184:
	br label %132

dead185:
	ret i32 0
}

define i32 @re_alt() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	store i32 -1, i32* %0
	store i32 0, i32* %1
	store i32 0, i32* %2
	store i32 0, i32* %3
	store i32 0, i32* %4
	br label %9

9:
	%10 = icmp ne i32 1, 0
	br i1 %10, label %11, label %23

11:
	%12 = load i32, i32* %2
	%13 = add i32 %12, 1
	store i32 %13, i32* %2
	%14 = call i32 @re_emit(i32 5, i32 0, i32 0)
	store i32 %14, i32* %3
	%15 = load i32, i32* @re_pc
	store i32 %15, i32* %4
	%16 = call i32 @re_cat()
	store i32 %16, i32* %5
	%17 = load i32, i32* %5
	%18 = icmp eq i32 %17, 0
	%19 = zext i1 %18 to i32
	%20 = icmp ne i32 %19, 0
	%21 = zext i1 %20 to i32
	%22 = icmp ne i32 %21, 0
	br i1 %22, label %28, label %34

23:
	%24 = load i32, i32* %3
	%25 = load i32, i32* %4
	%26 = call i32 @re_set(i32 %24, i32 1, i32 %25)
	%27 = load i32, i32* @re_pc
	store i32 %27, i32* %7
	br label %64

28:
	%29 = load i32, i32* @re_bad
	%30 = icmp eq i32 %29, 0
	%31 = zext i1 %30 to i32
	%32 = icmp ne i32 %31, 0
	%33 = zext i1 %32 to i32
	br label %34

34:
	%35 = phi i32 [ %21, %11 ], [ %33, %28 ]
	%36 = icmp ne i32 %35, 0
	br i1 %36, label %37, label %38

37:
	store i32 1, i32* %1
	br label %38

38:
	%39 = load i32, i32* @re_bad
	%40 = icmp ne i32 %39, 0
	%41 = zext i1 %40 to i32
	%42 = icmp ne i32 %41, 0
	br i1 %42, label %43, label %44

43:
	br label %23

44:
	%45 = call i32 @re_at(i32 0)
	%46 = icmp ne i32 %45, 124
	%47 = zext i1 %46 to i32
	%48 = icmp ne i32 %47, 0
	br i1 %48, label %49, label %50

dead201:
	br label %44

49:
	br label %23

50:
	%51 = load i32, i32* @re_pos
	%52 = add i32 %51, 1
	store i32 %52, i32* @re_pos
	%53 = load i32, i32* %0
	%54 = call i32 @re_emit(i32 5, i32 0, i32 %53)
	store i32 %54, i32* %6
	%55 = load i32, i32* %6
	store i32 %55, i32* %0
	%56 = load i32, i32* %3
	%57 = call i32 @re_set(i32 %56, i32 0, i32 4)
	%58 = load i32, i32* %3
	%59 = load i32, i32* %4
	%60 = call i32 @re_set(i32 %58, i32 1, i32 %59)
	%61 = load i32, i32* %3
	%62 = load i32, i32* @re_pc
	%63 = call i32 @re_set(i32 %61, i32 2, i32 %62)
	br label %9

dead202:
	br label %50

64:
	%65 = load i32, i32* %0
	%66 = icmp sge i32 %65, 0
	%67 = zext i1 %66 to i32
	%68 = icmp ne i32 %67, 0
	br i1 %68, label %69, label %76

69:
	%70 = load i32, i32* %0
	store i32 %70, i32* %8
	%71 = load i32, i32* %8
	%72 = call i32 @re_get(i32 %71, i32 2)
	store i32 %72, i32* %0
	%73 = load i32, i32* %8
	%74 = load i32, i32* %7
	%75 = call i32 @re_set(i32 %73, i32 1, i32 %74)
	br label %64

76:
	%77 = load i32, i32* %1
	%78 = icmp ne i32 %77, 0
	%79 = zext i1 %78 to i32
	%80 = load i32, i32* %2
	%81 = icmp sgt i32 %80, 1
	%82 = zext i1 %81 to i32
	%83 = load i32, i32* @re_depth
	%84 = icmp eq i32 %83, 0
	%85 = zext i1 %84 to i32
	%86 = or i32 %82, %85
	%87 = and i32 %79, %86
	%88 = load i32, i32* @re_bad
	%89 = icmp eq i32 %88, 0
	%90 = zext i1 %89 to i32
	%91 = and i32 %87, %90
	%92 = icmp ne i32 %91, 0
	br i1 %92, label %93, label %94

93:
	store i32 8, i32* @re_bad
	br label %94

94:
	ret i32 0

dead203:
	ret i32 0
}

define i32 @re_rep() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = alloca i32
	%15 = alloca i32
	%16 = alloca i32
	%17 = alloca i32
	%18 = alloca i32
	%19 = alloca i32
	%20 = alloca i32
	%21 = alloca i32
	%22 = alloca i32
	%23 = alloca i32
	%24 = alloca i32
	%25 = alloca i32
	%26 = alloca i32
	%27 = alloca i32
	%28 = alloca i32
	%29 = alloca i32
	%30 = alloca i32
	%31 = alloca i32
	%32 = alloca i32
	%33 = alloca i32
	%34 = alloca i32
	store i32 0, i32* %3
	store i32 0, i32* %4
	store i32 0, i32* %5
	store i32 -1, i32* %2
	%35 = call i32 @re_emit(i32 5, i32 0, i32 0)
	store i32 %35, i32* %8
	%36 = call i32 @re_emit(i32 5, i32 0, i32 0)
	store i32 %36, i32* %9
	%37 = load i32, i32* @re_ng
	store i32 %37, i32* %11
	%38 = call i32 @re_emit(i32 5, i32 0, i32 0)
	store i32 %38, i32* %10
	%39 = load i32, i32* %10
	store i32 %39, i32* %13
	%40 = call i32 @re_atom()
	%41 = load i32, i32* @re_pc
	store i32 %41, i32* %14
	%42 = load i32, i32* @re_ng
	store i32 %42, i32* %12
	%43 = load i32, i32* %12
	%44 = load i32, i32* %11
	%45 = icmp sgt i32 %43, %44
	%46 = zext i1 %45 to i32
	store i32 %46, i32* %15
	%47 = load i32, i32* %10
	%48 = load i32, i32* %15
	%49 = icmp ne i32 %48, 0
	br i1 %49, label %50, label %51

50:
	br label %52

51:
	br label %52

52:
	%53 = phi i32 [ 11, %50 ], [ 5, %51 ]
	%54 = call i32 @re_set(i32 %47, i32 0, i32 %53)
	%55 = load i32, i32* %10
	%56 = load i32, i32* %15
	%57 = icmp ne i32 %56, 0
	br i1 %57, label %58, label %62

58:
	%59 = load i32, i32* %11
	%60 = add i32 %59, 1
	%61 = mul i32 %60, 2
	br label %65

62:
	%63 = load i32, i32* %10
	%64 = add i32 %63, 1
	br label %65

65:
	%66 = phi i32 [ %61, %58 ], [ %64, %62 ]
	%67 = call i32 @re_set(i32 %55, i32 1, i32 %66)
	%68 = load i32, i32* %12
	%69 = mul i32 %68, 2
	%70 = add i32 %69, 1
	store i32 %70, i32* %16
	%71 = load i32, i32* %10
	%72 = load i32, i32* %16
	%73 = icmp slt i32 %72, 128
	%74 = zext i1 %73 to i32
	%75 = icmp ne i32 %74, 0
	br i1 %75, label %76, label %78

76:
	%77 = load i32, i32* %16
	br label %79

78:
	br label %79

79:
	%80 = phi i32 [ %77, %76 ], [ 127, %78 ]
	%81 = call i32 @re_set(i32 %71, i32 2, i32 %80)
	%82 = load i32, i32* @re_bad
	%83 = icmp ne i32 %82, 0
	%84 = zext i1 %83 to i32
	%85 = icmp ne i32 %84, 0
	br i1 %85, label %86, label %87

86:
	ret i32 0

87:
	%88 = call i32 @re_at(i32 0)
	store i32 %88, i32* %17
	%89 = call i32 @re_at(i32 1)
	store i32 %89, i32* %18
	%90 = load i32, i32* %17
	%91 = icmp eq i32 %90, 123
	%92 = zext i1 %91 to i32
	%93 = icmp ne i32 %92, 0
	%94 = zext i1 %93 to i32
	%95 = icmp ne i32 %94, 0
	br i1 %95, label %96, label %103

dead186:
	br label %87

96:
	%97 = load i32, i32* %18
	%98 = icmp sge i32 %97, 48
	%99 = zext i1 %98 to i32
	%100 = icmp ne i32 %99, 0
	%101 = zext i1 %100 to i32
	%102 = icmp ne i32 %101, 0
	br i1 %102, label %108, label %114

103:
	%104 = phi i32 [ %94, %87 ], [ %117, %114 ]
	%105 = icmp eq i32 %104, 0
	%106 = zext i1 %105 to i32
	%107 = icmp ne i32 %106, 0
	br i1 %107, label %118, label %124

108:
	%109 = load i32, i32* %18
	%110 = icmp sle i32 %109, 57
	%111 = zext i1 %110 to i32
	%112 = icmp ne i32 %111, 0
	%113 = zext i1 %112 to i32
	br label %114

114:
	%115 = phi i32 [ %101, %96 ], [ %113, %108 ]
	%116 = icmp ne i32 %115, 0
	%117 = zext i1 %116 to i32
	br label %103

118:
	%119 = call i32 @re_at(i32 0)
	store i32 %119, i32* %19
	store i32 0, i32* %20
	%120 = load i32, i32* %19
	%121 = icmp eq i32 %120, 42
	%122 = zext i1 %121 to i32
	%123 = icmp ne i32 %122, 0
	br i1 %123, label %133, label %139

124:
	%125 = load i32, i32* @re_pos
	%126 = add i32 %125, 1
	store i32 %126, i32* @re_pos
	%127 = call i32 @re_num()
	store i32 %127, i32* %24
	%128 = load i32, i32* %24
	store i32 %128, i32* %6
	%129 = call i32 @re_at(i32 0)
	%130 = icmp eq i32 %129, 44
	%131 = zext i1 %130 to i32
	%132 = icmp ne i32 %131, 0
	br i1 %132, label %234, label %246

133:
	%134 = load i32, i32* @re_pos
	%135 = add i32 %134, 1
	store i32 %135, i32* @re_pos
	store i32 1, i32* %3
	store i32 1, i32* %4
	store i32 1, i32* %5
	store i32 1, i32* %20
	br label %136

136:
	%137 = load i32, i32* %20
	%138 = icmp ne i32 %137, 0
	br i1 %138, label %157, label %162

139:
	%140 = load i32, i32* %19
	%141 = icmp eq i32 %140, 43
	%142 = zext i1 %141 to i32
	%143 = icmp ne i32 %142, 0
	br i1 %143, label %144, label %148

144:
	%145 = load i32, i32* @re_pos
	%146 = add i32 %145, 1
	store i32 %146, i32* @re_pos
	store i32 1, i32* %4
	store i32 1, i32* %5
	store i32 1, i32* %20
	br label %147

147:
	br label %136

148:
	%149 = load i32, i32* %19
	%150 = icmp eq i32 %149, 63
	%151 = zext i1 %150 to i32
	%152 = icmp ne i32 %151, 0
	br i1 %152, label %153, label %156

153:
	%154 = load i32, i32* @re_pos
	%155 = add i32 %154, 1
	store i32 %155, i32* @re_pos
	store i32 1, i32* %3
	store i32 1, i32* %5
	store i32 1, i32* %20
	br label %156

156:
	br label %147

157:
	%158 = call i32 @re_isq()
	%159 = icmp ne i32 %158, 0
	%160 = zext i1 %159 to i32
	%161 = icmp ne i32 %160, 0
	br i1 %161, label %167, label %168

162:
	%163 = load i32, i32* %5
	%164 = icmp eq i32 %163, 0
	%165 = zext i1 %164 to i32
	%166 = icmp ne i32 %165, 0
	br i1 %166, label %169, label %176

167:
	store i32 5, i32* @re_bad
	ret i32 0

168:
	br label %162

dead187:
	br label %168

169:
	%170 = load i32, i32* %8
	%171 = load i32, i32* %9
	%172 = call i32 @re_set(i32 %170, i32 1, i32 %171)
	%173 = load i32, i32* %9
	%174 = load i32, i32* %13
	%175 = call i32 @re_set(i32 %173, i32 1, i32 %174)
	ret i32 0

176:
	%177 = load i32, i32* %4
	%178 = icmp eq i32 %177, 0
	%179 = zext i1 %178 to i32
	%180 = icmp ne i32 %179, 0
	br i1 %180, label %181, label %193

dead188:
	br label %176

181:
	%182 = load i32, i32* %8
	%183 = call i32 @re_set(i32 %182, i32 0, i32 4)
	%184 = load i32, i32* %8
	%185 = load i32, i32* %9
	%186 = call i32 @re_set(i32 %184, i32 1, i32 %185)
	%187 = load i32, i32* %8
	%188 = load i32, i32* @re_pc
	%189 = call i32 @re_set(i32 %187, i32 2, i32 %188)
	%190 = load i32, i32* %9
	%191 = load i32, i32* %13
	%192 = call i32 @re_set(i32 %190, i32 1, i32 %191)
	ret i32 0

193:
	%194 = load i32, i32* %3
	%195 = icmp eq i32 %194, 0
	%196 = zext i1 %195 to i32
	%197 = icmp ne i32 %196, 0
	br i1 %197, label %198, label %215

dead189:
	br label %193

198:
	%199 = call i32 @re_mark()
	store i32 %199, i32* %21
	%200 = load i32, i32* %8
	%201 = load i32, i32* %9
	%202 = call i32 @re_set(i32 %200, i32 1, i32 %201)
	%203 = load i32, i32* %9
	%204 = call i32 @re_set(i32 %203, i32 0, i32 6)
	%205 = load i32, i32* %9
	%206 = load i32, i32* %21
	%207 = call i32 @re_set(i32 %205, i32 1, i32 %206)
	%208 = load i32, i32* %21
	%209 = call i32 @re_emit(i32 10, i32 %208, i32 0)
	%210 = load i32, i32* %9
	%211 = call i32 @re_emit(i32 4, i32 %210, i32 0)
	store i32 %211, i32* %22
	%212 = load i32, i32* %22
	%213 = load i32, i32* @re_pc
	%214 = call i32 @re_set(i32 %212, i32 2, i32 %213)
	ret i32 0

215:
	%216 = call i32 @re_mark()
	store i32 %216, i32* %23
	%217 = load i32, i32* %9
	%218 = call i32 @re_set(i32 %217, i32 0, i32 6)
	%219 = load i32, i32* %9
	%220 = load i32, i32* %23
	%221 = call i32 @re_set(i32 %219, i32 1, i32 %220)
	%222 = load i32, i32* %23
	%223 = call i32 @re_emit(i32 10, i32 %222, i32 0)
	%224 = load i32, i32* %8
	%225 = call i32 @re_emit(i32 5, i32 %224, i32 0)
	%226 = load i32, i32* %8
	%227 = call i32 @re_set(i32 %226, i32 0, i32 4)
	%228 = load i32, i32* %8
	%229 = load i32, i32* %9
	%230 = call i32 @re_set(i32 %228, i32 1, i32 %229)
	%231 = load i32, i32* %8
	%232 = load i32, i32* @re_pc
	%233 = call i32 @re_set(i32 %231, i32 2, i32 %232)
	ret i32 0

dead190:
	br label %215

dead191:
	br label %124

234:
	%235 = load i32, i32* @re_pos
	%236 = add i32 %235, 1
	store i32 %236, i32* @re_pos
	%237 = call i32 @re_at(i32 0)
	%238 = icmp eq i32 %237, 125
	%239 = zext i1 %238 to i32
	%240 = icmp ne i32 %239, 0
	br i1 %240, label %248, label %250

241:
	%242 = call i32 @re_at(i32 0)
	%243 = icmp ne i32 %242, 125
	%244 = zext i1 %243 to i32
	%245 = icmp ne i32 %244, 0
	br i1 %245, label %252, label %253

246:
	%247 = load i32, i32* %24
	store i32 %247, i32* %7
	br label %241

248:
	store i32 -1, i32* %7
	br label %249

249:
	br label %241

250:
	%251 = call i32 @re_num()
	store i32 %251, i32* %7
	br label %249

252:
	store i32 3, i32* @re_bad
	ret i32 0

253:
	%254 = load i32, i32* @re_pos
	%255 = add i32 %254, 1
	store i32 %255, i32* @re_pos
	%256 = load i32, i32* %7
	store i32 %256, i32* %25
	%257 = load i32, i32* %6
	store i32 %257, i32* %26
	%258 = load i32, i32* %25
	%259 = icmp sge i32 %258, 0
	%260 = zext i1 %259 to i32
	%261 = icmp ne i32 %260, 0
	%262 = zext i1 %261 to i32
	%263 = icmp ne i32 %262, 0
	br i1 %263, label %264, label %271

dead192:
	br label %253

264:
	%265 = load i32, i32* %25
	%266 = load i32, i32* %26
	%267 = icmp slt i32 %265, %266
	%268 = zext i1 %267 to i32
	%269 = icmp ne i32 %268, 0
	%270 = zext i1 %269 to i32
	br label %271

271:
	%272 = phi i32 [ %262, %253 ], [ %270, %264 ]
	%273 = icmp ne i32 %272, 0
	br i1 %273, label %274, label %275

274:
	store i32 3, i32* @re_bad
	ret i32 0

275:
	%276 = call i32 @re_isq()
	%277 = icmp ne i32 %276, 0
	%278 = zext i1 %277 to i32
	%279 = icmp ne i32 %278, 0
	br i1 %279, label %280, label %281

dead193:
	br label %275

280:
	store i32 5, i32* @re_bad
	ret i32 0

281:
	%282 = load i32, i32* %25
	%283 = icmp eq i32 %282, 0
	%284 = zext i1 %283 to i32
	%285 = icmp ne i32 %284, 0
	%286 = zext i1 %285 to i32
	%287 = icmp ne i32 %286, 0
	br i1 %287, label %288, label %294

dead194:
	br label %281

288:
	%289 = load i32, i32* %26
	%290 = icmp eq i32 %289, 0
	%291 = zext i1 %290 to i32
	%292 = icmp ne i32 %291, 0
	%293 = zext i1 %292 to i32
	br label %294

294:
	%295 = phi i32 [ %286, %281 ], [ %293, %288 ]
	%296 = icmp ne i32 %295, 0
	br i1 %296, label %297, label %301

297:
	%298 = load i32, i32* %8
	%299 = load i32, i32* %14
	%300 = call i32 @re_set(i32 %298, i32 1, i32 %299)
	ret i32 0

301:
	%302 = load i32, i32* %25
	%303 = icmp slt i32 %302, 0
	%304 = zext i1 %303 to i32
	%305 = icmp ne i32 %304, 0
	br i1 %305, label %306, label %311

dead195:
	br label %301

306:
	%307 = load i32, i32* %26
	%308 = icmp eq i32 %307, 0
	%309 = zext i1 %308 to i32
	%310 = icmp ne i32 %309, 0
	br i1 %310, label %316, label %335

311:
	%312 = load i32, i32* %26
	%313 = icmp eq i32 %312, 0
	%314 = zext i1 %313 to i32
	%315 = icmp ne i32 %314, 0
	br i1 %315, label %384, label %399

316:
	%317 = call i32 @re_mark()
	store i32 %317, i32* %27
	%318 = load i32, i32* %9
	%319 = call i32 @re_set(i32 %318, i32 0, i32 6)
	%320 = load i32, i32* %9
	%321 = load i32, i32* %27
	%322 = call i32 @re_set(i32 %320, i32 1, i32 %321)
	%323 = load i32, i32* %27
	%324 = call i32 @re_emit(i32 10, i32 %323, i32 0)
	%325 = load i32, i32* %8
	%326 = call i32 @re_emit(i32 5, i32 %325, i32 0)
	%327 = load i32, i32* %8
	%328 = call i32 @re_set(i32 %327, i32 0, i32 4)
	%329 = load i32, i32* %8
	%330 = load i32, i32* %9
	%331 = call i32 @re_set(i32 %329, i32 1, i32 %330)
	%332 = load i32, i32* %8
	%333 = load i32, i32* @re_pc
	%334 = call i32 @re_set(i32 %332, i32 2, i32 %333)
	ret i32 0

335:
	%336 = load i32, i32* %8
	%337 = load i32, i32* %9
	%338 = call i32 @re_set(i32 %336, i32 1, i32 %337)
	%339 = load i32, i32* %9
	%340 = load i32, i32* %13
	%341 = call i32 @re_set(i32 %339, i32 1, i32 %340)
	store i32 1, i32* %0
	br label %342

dead196:
	br label %335

342:
	%343 = load i32, i32* %0
	%344 = load i32, i32* %26
	%345 = icmp slt i32 %343, %344
	%346 = zext i1 %345 to i32
	%347 = icmp ne i32 %346, 0
	%348 = zext i1 %347 to i32
	%349 = icmp ne i32 %348, 0
	br i1 %349, label %375, label %381

350:
	%351 = load i32, i32* %13
	%352 = load i32, i32* %14
	%353 = call i32 @re_copy(i32 %351, i32 %352)
	br label %354

354:
	%355 = load i32, i32* %0
	%356 = add i32 %355, 1
	store i32 %356, i32* %0
	br label %342

357:
	%358 = call i32 @re_emit(i32 4, i32 0, i32 0)
	store i32 %358, i32* %29
	%359 = call i32 @re_mark()
	store i32 %359, i32* %28
	%360 = load i32, i32* %28
	%361 = call i32 @re_emit(i32 6, i32 %360, i32 0)
	store i32 %361, i32* %30
	%362 = load i32, i32* %13
	%363 = load i32, i32* %14
	%364 = call i32 @re_copy(i32 %362, i32 %363)
	%365 = load i32, i32* %28
	%366 = call i32 @re_emit(i32 10, i32 %365, i32 0)
	%367 = load i32, i32* %29
	%368 = call i32 @re_emit(i32 5, i32 %367, i32 0)
	%369 = load i32, i32* %29
	%370 = load i32, i32* %30
	%371 = call i32 @re_set(i32 %369, i32 1, i32 %370)
	%372 = load i32, i32* %29
	%373 = load i32, i32* @re_pc
	%374 = call i32 @re_set(i32 %372, i32 2, i32 %373)
	ret i32 0

375:
	%376 = load i32, i32* @re_bad
	%377 = icmp eq i32 %376, 0
	%378 = zext i1 %377 to i32
	%379 = icmp ne i32 %378, 0
	%380 = zext i1 %379 to i32
	br label %381

381:
	%382 = phi i32 [ %348, %342 ], [ %380, %375 ]
	%383 = icmp ne i32 %382, 0
	br i1 %383, label %350, label %357

dead197:
	br label %311

384:
	%385 = load i32, i32* %8
	%386 = call i32 @re_set(i32 %385, i32 0, i32 4)
	%387 = load i32, i32* %8
	%388 = load i32, i32* %9
	%389 = call i32 @re_set(i32 %387, i32 1, i32 %388)
	%390 = load i32, i32* %8
	%391 = load i32, i32* %2
	%392 = call i32 @re_set(i32 %390, i32 2, i32 %391)
	%393 = load i32, i32* %8
	store i32 %393, i32* %2
	%394 = load i32, i32* %9
	%395 = load i32, i32* %13
	%396 = call i32 @re_set(i32 %394, i32 1, i32 %395)
	store i32 1, i32* %0
	br label %406

397:
	%398 = load i32, i32* @re_pc
	store i32 %398, i32* %33
	br label %496

399:
	%400 = load i32, i32* %8
	%401 = load i32, i32* %9
	%402 = call i32 @re_set(i32 %400, i32 1, i32 %401)
	%403 = load i32, i32* %9
	%404 = load i32, i32* %13
	%405 = call i32 @re_set(i32 %403, i32 1, i32 %404)
	store i32 1, i32* %0
	br label %438

406:
	%407 = load i32, i32* %0
	%408 = load i32, i32* %25
	%409 = icmp slt i32 %407, %408
	%410 = zext i1 %409 to i32
	%411 = icmp ne i32 %410, 0
	%412 = zext i1 %411 to i32
	%413 = icmp ne i32 %412, 0
	br i1 %413, label %429, label %435

414:
	%415 = load i32, i32* %2
	%416 = call i32 @re_emit(i32 4, i32 0, i32 %415)
	store i32 %416, i32* %31
	%417 = load i32, i32* %31
	store i32 %417, i32* %2
	%418 = load i32, i32* @re_pc
	store i32 %418, i32* %32
	%419 = load i32, i32* %13
	%420 = load i32, i32* %14
	%421 = call i32 @re_copy(i32 %419, i32 %420)
	%422 = load i32, i32* %31
	%423 = load i32, i32* %32
	%424 = call i32 @re_set(i32 %422, i32 1, i32 %423)
	br label %425

425:
	%426 = load i32, i32* %0
	%427 = add i32 %426, 1
	store i32 %427, i32* %0
	br label %406

428:
	br label %397

429:
	%430 = load i32, i32* @re_bad
	%431 = icmp eq i32 %430, 0
	%432 = zext i1 %431 to i32
	%433 = icmp ne i32 %432, 0
	%434 = zext i1 %433 to i32
	br label %435

435:
	%436 = phi i32 [ %412, %406 ], [ %434, %429 ]
	%437 = icmp ne i32 %436, 0
	br i1 %437, label %414, label %428

438:
	%439 = load i32, i32* %0
	%440 = load i32, i32* %26
	%441 = icmp slt i32 %439, %440
	%442 = zext i1 %441 to i32
	%443 = icmp ne i32 %442, 0
	%444 = zext i1 %443 to i32
	%445 = icmp ne i32 %444, 0
	br i1 %445, label %455, label %461

446:
	%447 = load i32, i32* %13
	%448 = load i32, i32* %14
	%449 = call i32 @re_copy(i32 %447, i32 %448)
	br label %450

450:
	%451 = load i32, i32* %0
	%452 = add i32 %451, 1
	store i32 %452, i32* %0
	br label %438

453:
	%454 = load i32, i32* %26
	store i32 %454, i32* %1
	br label %464

455:
	%456 = load i32, i32* @re_bad
	%457 = icmp eq i32 %456, 0
	%458 = zext i1 %457 to i32
	%459 = icmp ne i32 %458, 0
	%460 = zext i1 %459 to i32
	br label %461

461:
	%462 = phi i32 [ %444, %438 ], [ %460, %455 ]
	%463 = icmp ne i32 %462, 0
	br i1 %463, label %446, label %453

464:
	%465 = load i32, i32* %1
	%466 = load i32, i32* %25
	%467 = icmp slt i32 %465, %466
	%468 = zext i1 %467 to i32
	%469 = icmp ne i32 %468, 0
	%470 = zext i1 %469 to i32
	%471 = icmp ne i32 %470, 0
	br i1 %471, label %487, label %493

472:
	%473 = load i32, i32* %2
	%474 = call i32 @re_emit(i32 4, i32 0, i32 %473)
	store i32 %474, i32* %31
	%475 = load i32, i32* %31
	store i32 %475, i32* %2
	%476 = load i32, i32* @re_pc
	store i32 %476, i32* %32
	%477 = load i32, i32* %13
	%478 = load i32, i32* %14
	%479 = call i32 @re_copy(i32 %477, i32 %478)
	%480 = load i32, i32* %31
	%481 = load i32, i32* %32
	%482 = call i32 @re_set(i32 %480, i32 1, i32 %481)
	br label %483

483:
	%484 = load i32, i32* %1
	%485 = add i32 %484, 1
	store i32 %485, i32* %1
	br label %464

486:
	br label %397

487:
	%488 = load i32, i32* @re_bad
	%489 = icmp eq i32 %488, 0
	%490 = zext i1 %489 to i32
	%491 = icmp ne i32 %490, 0
	%492 = zext i1 %491 to i32
	br label %493

493:
	%494 = phi i32 [ %470, %464 ], [ %492, %487 ]
	%495 = icmp ne i32 %494, 0
	br i1 %495, label %472, label %486

496:
	%497 = load i32, i32* %2
	%498 = icmp sge i32 %497, 0
	%499 = zext i1 %498 to i32
	%500 = icmp ne i32 %499, 0
	br i1 %500, label %501, label %508

501:
	%502 = load i32, i32* %2
	store i32 %502, i32* %34
	%503 = load i32, i32* %34
	%504 = call i32 @re_get(i32 %503, i32 2)
	store i32 %504, i32* %2
	%505 = load i32, i32* %34
	%506 = load i32, i32* %33
	%507 = call i32 @re_set(i32 %505, i32 2, i32 %506)
	br label %496

508:
	ret i32 0

dead198:
	ret i32 0
}

define i32 @re_cat() {
entry:
	%0 = alloca i32
	%1 = alloca i32
	%2 = alloca i32
	%3 = alloca i32
	store i32 0, i32* %0
	br label %4

4:
	%5 = icmp ne i32 1, 0
	br i1 %5, label %6, label %30

6:
	%7 = call i32 @re_at(i32 0)
	store i32 %7, i32* %1
	%8 = load i32, i32* %1
	%9 = icmp eq i32 %8, 41
	%10 = zext i1 %9 to i32
	%11 = load i32, i32* @re_depth
	%12 = icmp sgt i32 %11, 0
	%13 = zext i1 %12 to i32
	%14 = and i32 %10, %13
	store i32 %14, i32* %2
	%15 = load i32, i32* %1
	%16 = icmp eq i32 %15, 0
	%17 = zext i1 %16 to i32
	%18 = load i32, i32* %1
	%19 = icmp eq i32 %18, 124
	%20 = zext i1 %19 to i32
	%21 = or i32 %17, %20
	%22 = load i32, i32* %2
	%23 = or i32 %21, %22
	store i32 %23, i32* %3
	%24 = load i32, i32* %3
	%25 = load i32, i32* @re_bad
	%26 = icmp ne i32 %25, 0
	%27 = zext i1 %26 to i32
	%28 = or i32 %24, %27
	%29 = icmp ne i32 %28, 0
	br i1 %29, label %32, label %33

30:
	%31 = load i32, i32* %0
	ret i32 %31

32:
	br label %30

33:
	%34 = call i32 @re_rep()
	%35 = load i32, i32* %0
	%36 = add i32 %35, 1
	store i32 %36, i32* %0
	br label %4

dead199:
	br label %33

dead200:
	ret i32 0
}

define i32 @re_run(i32 %0, i32 %1) {
entry:
	%2 = alloca i32
	store i32 %0, i32* %2
	%3 = alloca i32
	store i32 %1, i32* %3
	%4 = alloca i32
	%5 = alloca i32
	%6 = alloca i32
	%7 = alloca i32
	%8 = alloca i32
	%9 = alloca i32
	%10 = alloca i32
	%11 = alloca i32*
	%12 = alloca i32
	%13 = alloca i32
	%14 = alloca i32
	%15 = alloca i32
	%16 = alloca i32
	%17 = alloca i32
	%18 = alloca i32
	%19 = alloca i32
	%20 = alloca i32
	%21 = alloca i32
	%22 = alloca i32
	%23 = alloca i32
	%24 = alloca i32
	%25 = alloca i32
	%26 = load i32, i32* %2
	store i32 %26, i32* %4
	%27 = load i32, i32* %3
	store i32 %27, i32* %5
	br label %28

28:
	%29 = icmp ne i32 1, 0
	br i1 %29, label %30, label %36

30:
	%31 = load i32, i32* @re_steps
	store i32 %31, i32* %6
	%32 = load i32, i32* %6
	%33 = icmp sge i32 %32, 400000
	%34 = zext i1 %33 to i32
	%35 = icmp ne i32 %34, 0
	br i1 %35, label %37, label %38

36:
	ret i32 0

37:
	ret i32 -2

38:
	%39 = load i32, i32* %6
	%40 = add i32 %39, 1
	store i32 %40, i32* @re_steps
	%41 = load i32, i32* %4
	%42 = call i32 @re_get(i32 %41, i32 0)
	store i32 %42, i32* %7
	%43 = load i32, i32* %4
	%44 = call i32 @re_get(i32 %43, i32 1)
	store i32 %44, i32* %8
	%45 = load i32, i32* %4
	%46 = call i32 @re_get(i32 %45, i32 2)
	store i32 %46, i32* %9
	%47 = load i32, i32* @re_slen
	store i32 %47, i32* %10
	%48 = load i32*, i32** @re_subj
	%49 = bitcast i32* %48 to i32*
	store i32* %49, i32** %11
	%50 = load i32, i32* %5
	%51 = load i32, i32* %10
	%52 = icmp sge i32 %50, %51
	%53 = zext i1 %52 to i32
	store i32 %53, i32* %12
	%54 = load i32, i32* %4
	%55 = add i32 %54, 1
	store i32 %55, i32* %13
	%56 = load i32, i32* %5
	%57 = add i32 %56, 1
	store i32 %57, i32* %14
	%58 = load i32, i32* %7
	%59 = icmp eq i32 %58, 1
	%60 = zext i1 %59 to i32
	%61 = icmp ne i32 %60, 0
	br i1 %61, label %62, label %65

dead204:
	br label %38

62:
	%63 = load i32, i32* %12
	%64 = icmp ne i32 %63, 0
	br i1 %64, label %70, label %71

65:
	%66 = load i32, i32* %7
	%67 = icmp eq i32 %66, 2
	%68 = zext i1 %67 to i32
	%69 = icmp ne i32 %68, 0
	br i1 %69, label %88, label %91

70:
	ret i32 0

71:
	%72 = load i32, i32* %5
	%73 = load i32*, i32** %11
	%74 = getelementptr i8, i32* %73, i32 %72
	%75 = load i8, i8* %74
	%76 = sext i8 %75 to i32
	%77 = and i32 %76, 255
	%78 = call i32 @re_fold1(i32 %77)
	store i32 %78, i32* %15
	%79 = load i32, i32* %15
	%80 = load i32, i32* %8
	%81 = icmp ne i32 %79, %80
	%82 = zext i1 %81 to i32
	%83 = icmp ne i32 %82, 0
	br i1 %83, label %84, label %85

dead205:
	br label %71

84:
	ret i32 0

85:
	%86 = load i32, i32* %13
	store i32 %86, i32* %4
	%87 = load i32, i32* %14
	store i32 %87, i32* %5
	br label %28

dead206:
	br label %85

dead207:
	br label %65

88:
	%89 = load i32, i32* %12
	%90 = icmp ne i32 %89, 0
	br i1 %90, label %96, label %97

91:
	%92 = load i32, i32* %7
	%93 = icmp eq i32 %92, 3
	%94 = zext i1 %93 to i32
	%95 = icmp ne i32 %94, 0
	br i1 %95, label %100, label %103

96:
	ret i32 0

97:
	%98 = load i32, i32* %13
	store i32 %98, i32* %4
	%99 = load i32, i32* %14
	store i32 %99, i32* %5
	br label %28

dead208:
	br label %97

dead209:
	br label %91

100:
	%101 = load i32, i32* %12
	%102 = icmp ne i32 %101, 0
	br i1 %102, label %108, label %109

103:
	%104 = load i32, i32* %7
	%105 = icmp eq i32 %104, 4
	%106 = zext i1 %105 to i32
	%107 = icmp ne i32 %106, 0
	br i1 %107, label %139, label %147

108:
	ret i32 0

109:
	%110 = load i32, i32* %5
	%111 = load i32*, i32** %11
	%112 = getelementptr i8, i32* %111, i32 %110
	%113 = load i8, i8* %112
	%114 = sext i8 %113 to i32
	%115 = and i32 %114, 255
	%116 = call i32 @re_fold1(i32 %115)
	store i32 %116, i32* %16
	%117 = load i32, i32* %8
	%118 = mul i32 %117, 32
	%119 = load i32, i32* %16
	%120 = sdiv i32 %119, 8
	%121 = add i32 %118, %120
	%122 = getelementptr [2048 x i8], [2048 x i8]* @re_cls, i32 0, i32 %121
	%123 = load i8, i8* %122
	%124 = zext i8 %123 to i32
	%125 = and i32 %124, 255
	store i32 %125, i32* %17
	%126 = load i32, i32* %17
	%127 = load i32, i32* %16
	%128 = srem i32 %127, 8
	%129 = ashr i32 %126, %128
	%130 = and i32 %129, 1
	store i32 %130, i32* %18
	%131 = load i32, i32* %18
	%132 = icmp eq i32 %131, 0
	%133 = zext i1 %132 to i32
	%134 = icmp ne i32 %133, 0
	br i1 %134, label %135, label %136

dead210:
	br label %109

135:
	ret i32 0

136:
	%137 = load i32, i32* %13
	store i32 %137, i32* %4
	%138 = load i32, i32* %14
	store i32 %138, i32* %5
	br label %28

dead211:
	br label %136

dead212:
	br label %103

139:
	%140 = load i32, i32* %8
	%141 = load i32, i32* %5
	%142 = call i32 @re_run(i32 %140, i32 %141)
	store i32 %142, i32* %19
	%143 = load i32, i32* %19
	%144 = icmp ne i32 %143, 0
	%145 = zext i1 %144 to i32
	%146 = icmp ne i32 %145, 0
	br i1 %146, label %152, label %154

147:
	%148 = load i32, i32* %7
	%149 = icmp eq i32 %148, 5
	%150 = zext i1 %149 to i32
	%151 = icmp ne i32 %150, 0
	br i1 %151, label %156, label %158

152:
	%153 = load i32, i32* %19
	ret i32 %153

154:
	%155 = load i32, i32* %9
	store i32 %155, i32* %4
	br label %28

dead213:
	br label %154

dead214:
	br label %147

156:
	%157 = load i32, i32* %8
	store i32 %157, i32* %4
	br label %28

158:
	%159 = load i32, i32* %7
	%160 = icmp eq i32 %159, 6
	%161 = zext i1 %160 to i32
	%162 = icmp ne i32 %161, 0
	br i1 %162, label %163, label %177

dead215:
	br label %158

163:
	%164 = load i32, i32* %8
	%165 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %164
	%166 = load i32, i32* %165
	store i32 %166, i32* %20
	%167 = load i32, i32* %8
	%168 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %167
	%169 = load i32, i32* %5
	store i32 %169, i32* %168
	%170 = load i32, i32* %13
	%171 = load i32, i32* %5
	%172 = call i32 @re_run(i32 %170, i32 %171)
	store i32 %172, i32* %21
	%173 = load i32, i32* %21
	%174 = icmp eq i32 %173, 0
	%175 = zext i1 %174 to i32
	%176 = icmp ne i32 %175, 0
	br i1 %176, label %182, label %186

177:
	%178 = load i32, i32* %7
	%179 = icmp eq i32 %178, 7
	%180 = zext i1 %179 to i32
	%181 = icmp ne i32 %180, 0
	br i1 %181, label %188, label %189

182:
	%183 = load i32, i32* %8
	%184 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %183
	%185 = load i32, i32* %20
	store i32 %185, i32* %184
	br label %186

186:
	%187 = load i32, i32* %21
	ret i32 %187

dead216:
	br label %177

188:
	ret i32 1

189:
	%190 = load i32, i32* %7
	%191 = icmp eq i32 %190, 8
	%192 = zext i1 %191 to i32
	%193 = icmp ne i32 %192, 0
	br i1 %193, label %194, label %199

dead217:
	br label %189

194:
	%195 = load i32, i32* %5
	%196 = icmp ne i32 %195, 0
	%197 = zext i1 %196 to i32
	%198 = icmp ne i32 %197, 0
	br i1 %198, label %204, label %205

199:
	%200 = load i32, i32* %7
	%201 = icmp eq i32 %200, 9
	%202 = zext i1 %201 to i32
	%203 = icmp ne i32 %202, 0
	br i1 %203, label %207, label %213

204:
	ret i32 0

205:
	%206 = load i32, i32* %13
	store i32 %206, i32* %4
	br label %28

dead218:
	br label %205

dead219:
	br label %199

207:
	%208 = load i32, i32* %5
	%209 = load i32, i32* %10
	%210 = icmp ne i32 %208, %209
	%211 = zext i1 %210 to i32
	%212 = icmp ne i32 %211, 0
	br i1 %212, label %218, label %219

213:
	%214 = load i32, i32* %7
	%215 = icmp eq i32 %214, 10
	%216 = zext i1 %215 to i32
	%217 = icmp ne i32 %216, 0
	br i1 %217, label %221, label %229

218:
	ret i32 0

219:
	%220 = load i32, i32* %13
	store i32 %220, i32* %4
	br label %28

dead220:
	br label %219

dead221:
	br label %213

221:
	%222 = load i32, i32* %5
	%223 = load i32, i32* %8
	%224 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %223
	%225 = load i32, i32* %224
	%226 = icmp eq i32 %222, %225
	%227 = zext i1 %226 to i32
	%228 = icmp ne i32 %227, 0
	br i1 %228, label %234, label %235

229:
	%230 = load i32, i32* %7
	%231 = icmp eq i32 %230, 11
	%232 = zext i1 %231 to i32
	%233 = icmp ne i32 %232, 0
	br i1 %233, label %237, label %249

234:
	ret i32 0

235:
	%236 = load i32, i32* %13
	store i32 %236, i32* %4
	br label %28

dead222:
	br label %235

dead223:
	br label %229

237:
	%238 = load i32, i32* @re_ctop
	store i32 %238, i32* %22
	%239 = load i32, i32* %9
	%240 = load i32, i32* %8
	%241 = sub i32 %239, %240
	%242 = add i32 %241, 1
	store i32 %242, i32* %23
	%243 = load i32, i32* %22
	%244 = load i32, i32* %23
	%245 = add i32 %243, %244
	%246 = icmp sgt i32 %245, u0x2000
	%247 = zext i1 %246 to i32
	%248 = icmp ne i32 %247, 0
	br i1 %248, label %250, label %251

249:
	ret i32 0

250:
	ret i32 0

251:
	%252 = load i32, i32* %8
	store i32 %252, i32* %25
	br label %253

dead224:
	br label %251

253:
	%254 = load i32, i32* %25
	%255 = load i32, i32* %9
	%256 = add i32 %255, 1
	%257 = icmp slt i32 %254, %256
	%258 = zext i1 %257 to i32
	%259 = icmp ne i32 %258, 0
	br i1 %259, label %260, label %275

260:
	%261 = load i32, i32* %22
	%262 = load i32, i32* %25
	%263 = load i32, i32* %8
	%264 = sub i32 %262, %263
	%265 = add i32 %261, %264
	%266 = getelementptr [8192 x i32], [8192 x i32]* @re_cstk, i32 0, i32 %265
	%267 = load i32, i32* %25
	%268 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %267
	%269 = load i32, i32* %268
	store i32 %269, i32* %266
	%270 = load i32, i32* %25
	%271 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %270
	store i32 -1, i32* %271
	br label %272

272:
	%273 = load i32, i32* %25
	%274 = add i32 %273, 1
	store i32 %274, i32* %25
	br label %253

275:
	%276 = load i32, i32* %22
	%277 = load i32, i32* %23
	%278 = add i32 %276, %277
	store i32 %278, i32* @re_ctop
	%279 = load i32, i32* %13
	%280 = load i32, i32* %5
	%281 = call i32 @re_run(i32 %279, i32 %280)
	store i32 %281, i32* %24
	%282 = load i32, i32* %22
	store i32 %282, i32* @re_ctop
	%283 = load i32, i32* %24
	%284 = icmp eq i32 %283, 0
	%285 = zext i1 %284 to i32
	%286 = icmp ne i32 %285, 0
	br i1 %286, label %287, label %289

287:
	%288 = load i32, i32* %8
	store i32 %288, i32* %25
	br label %291

289:
	%290 = load i32, i32* %24
	ret i32 %290

291:
	%292 = load i32, i32* %25
	%293 = load i32, i32* %9
	%294 = add i32 %293, 1
	%295 = icmp slt i32 %292, %294
	%296 = zext i1 %295 to i32
	%297 = icmp ne i32 %296, 0
	br i1 %297, label %298, label %311

298:
	%299 = load i32, i32* %25
	%300 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %299
	%301 = load i32, i32* %22
	%302 = load i32, i32* %25
	%303 = load i32, i32* %8
	%304 = sub i32 %302, %303
	%305 = add i32 %301, %304
	%306 = getelementptr [8192 x i32], [8192 x i32]* @re_cstk, i32 0, i32 %305
	%307 = load i32, i32* %306
	store i32 %307, i32* %300
	br label %308

308:
	%309 = load i32, i32* %25
	%310 = add i32 %309, 1
	store i32 %310, i32* %25
	br label %291

311:
	br label %289

dead225:
	br label %249

dead226:
	br label %28

dead227:
	ret i32 0
}

define i32 @rt_regex_search(i32* %0, i32* %1, i32* %2, i32* %3, i32 %4) {
entry:
	%5 = alloca i32*
	store i32* %0, i32** %5
	%6 = alloca i32*
	store i32* %1, i32** %6
	%7 = alloca i32*
	store i32* %2, i32** %7
	%8 = alloca i32*
	store i32* %3, i32** %8
	%9 = alloca i32
	store i32 %4, i32* %9
	%10 = alloca i32
	%11 = alloca i32
	%12 = alloca i32
	%13 = alloca i32
	%14 = alloca i32
	%15 = alloca i32
	%16 = alloca i32
	%17 = alloca i32*
	%18 = alloca i32
	%19 = alloca i32
	%20 = alloca i32
	%21 = alloca i32
	%22 = alloca i32
	%23 = alloca i32
	%24 = load i32*, i32** %8
	%25 = bitcast i32* %24 to i32*
	%26 = inttoptr i32 0 to i32*
	%27 = icmp ne i32* %25, %26
	%28 = zext i1 %27 to i32
	store i32 %28, i32* %14
	%29 = load i32, i32* %14
	%30 = icmp ne i32 %29, 0
	br i1 %30, label %31, label %32

31:
	store i32 0, i32* %10
	br label %39

32:
	store i32 0, i32* @re_fold
	%33 = load i32*, i32** %7
	%34 = bitcast i32* %33 to i32*
	%35 = inttoptr i32 0 to i32*
	%36 = icmp ne i32* %34, %35
	%37 = zext i1 %36 to i32
	%38 = icmp ne i32 %37, 0
	br i1 %38, label %53, label %54

39:
	%40 = load i32, i32* %10
	%41 = load i32, i32* %9
	%42 = icmp slt i32 %40, %41
	%43 = zext i1 %42 to i32
	%44 = icmp ne i32 %43, 0
	br i1 %44, label %45, label %52

45:
	%46 = load i32, i32* %10
	%47 = load i32*, i32** %8
	%48 = getelementptr i32, i32* %47, i32 %46
	store i32 -1, i32* %48
	br label %49

49:
	%50 = load i32, i32* %10
	%51 = add i32 %50, 1
	store i32 %51, i32* %10
	br label %39

52:
	br label %32

53:
	store i32 0, i32* %11
	br label %61

54:
	%55 = load i32*, i32** %5
	%56 = bitcast i32* %55 to i32*
	%57 = inttoptr i32 0 to i32*
	%58 = icmp eq i32* %56, %57
	%59 = zext i1 %58 to i32
	%60 = icmp ne i32 %59, 0
	br i1 %60, label %85, label %95

61:
	%62 = icmp ne i32 1, 0
	br i1 %62, label %63, label %74

63:
	%64 = load i32, i32* %11
	%65 = load i32*, i32** %7
	%66 = getelementptr i8, i32* %65, i32 %64
	%67 = load i8, i8* %66
	%68 = sext i8 %67 to i32
	%69 = and i32 %68, 255
	store i32 %69, i32* %15
	%70 = load i32, i32* %15
	%71 = icmp eq i32 %70, 0
	%72 = zext i1 %71 to i32
	%73 = icmp ne i32 %72, 0
	br i1 %73, label %75, label %76

74:
	br label %54

75:
	br label %74

76:
	%77 = load i32, i32* %15
	%78 = icmp eq i32 %77, 105
	%79 = zext i1 %78 to i32
	%80 = icmp ne i32 %79, 0
	br i1 %80, label %81, label %82

dead228:
	br label %76

81:
	store i32 1, i32* @re_fold
	br label %82

82:
	%83 = load i32, i32* %11
	%84 = add i32 %83, 1
	store i32 %84, i32* %11
	br label %61

85:
	%86 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%87 = bitcast i8* %86 to i32*
	store i32* %87, i32** @re_pat
	br label %88

88:
	store i32 0, i32* @re_pos
	store i32 0, i32* @re_pc
	store i32 0, i32* @re_ng
	store i32 0, i32* @re_ncls
	store i32 0, i32* @re_nmark
	store i32 0, i32* @re_bad
	store i32 0, i32* @re_depth
	%89 = call i32 @re_emit(i32 6, i32 0, i32 0)
	%90 = call i32 @re_alt()
	%91 = load i32, i32* @re_bad
	%92 = icmp ne i32 %91, 0
	%93 = zext i1 %92 to i32
	%94 = icmp ne i32 %93, 0
	br i1 %94, label %98, label %106

95:
	%96 = load i32*, i32** %5
	%97 = bitcast i32* %96 to i32*
	store i32* %97, i32** @re_pat
	br label %88

98:
	%99 = load i32, i32* @re_bad
	store i32 %99, i32* %16
	%100 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%101 = bitcast i8* %100 to i32*
	store i32* %101, i32** %17
	%102 = load i32, i32* %16
	%103 = icmp eq i32 %102, 1
	%104 = zext i1 %103 to i32
	%105 = icmp ne i32 %104, 0
	br i1 %105, label %119, label %140

106:
	%107 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%108 = bitcast i8* %107 to i32*
	store i32* %108, i32** @re_err
	%109 = call i32 @re_emit(i32 6, i32 1, i32 0)
	%110 = call i32 @re_emit(i32 7, i32 0, i32 0)
	%111 = load i32*, i32** %6
	%112 = bitcast i32* %111 to i32*
	%113 = call i32 @rt_strlen(i32* %112)
	store i32 %113, i32* %18
	%114 = load i32*, i32** %6
	%115 = bitcast i32* %114 to i32*
	store i32* %115, i32** @re_subj
	%116 = load i32, i32* %18
	store i32 %116, i32* @re_slen
	store i32 0, i32* @re_steps
	store i32 0, i32* @re_ctop
	%117 = load i32, i32* @re_ng
	%118 = add i32 %117, 1
	store i32 %118, i32* %19
	store i32 0, i32* %13
	br label %414

119:
	%120 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 0
	store i8 85, i8* %120
	%121 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 1
	store i8 110, i8* %121
	%122 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 2
	store i8 109, i8* %122
	%123 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 3
	store i8 97, i8* %123
	%124 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 4
	store i8 116, i8* %124
	%125 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 5
	store i8 99, i8* %125
	%126 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 6
	store i8 104, i8* %126
	%127 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 7
	store i8 101, i8* %127
	%128 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 8
	store i8 100, i8* %128
	%129 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 9
	store i8 32, i8* %129
	%130 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 10
	store i8 40, i8* %130
	%131 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 11
	store i8 32, i8* %131
	%132 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 12
	store i8 111, i8* %132
	%133 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 13
	store i8 114, i8* %133
	%134 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 14
	store i8 32, i8* %134
	%135 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 15
	store i8 92, i8* %135
	%136 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 16
	store i8 40, i8* %136
	%137 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 17
	store i8 0, i8* %137
	%138 = getelementptr [18 x i8], [18 x i8]* @.str.45, i32 0, i32 0
	%139 = bitcast i8* %138 to i32*
	store i32* %139, i32** %17
	br label %140

140:
	%141 = load i32, i32* %16
	%142 = icmp eq i32 %141, 2
	%143 = zext i1 %142 to i32
	%144 = icmp ne i32 %143, 0
	br i1 %144, label %145, label %179

145:
	%146 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 0
	store i8 85, i8* %146
	%147 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 1
	store i8 110, i8* %147
	%148 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 2
	store i8 109, i8* %148
	%149 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 3
	store i8 97, i8* %149
	%150 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 4
	store i8 116, i8* %150
	%151 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 5
	store i8 99, i8* %151
	%152 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 6
	store i8 104, i8* %152
	%153 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 7
	store i8 101, i8* %153
	%154 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 8
	store i8 100, i8* %154
	%155 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 9
	store i8 32, i8* %155
	%156 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 10
	store i8 91, i8* %156
	%157 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 11
	store i8 44, i8* %157
	%158 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 12
	store i8 32, i8* %158
	%159 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 13
	store i8 91, i8* %159
	%160 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 14
	store i8 94, i8* %160
	%161 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 15
	store i8 44, i8* %161
	%162 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 16
	store i8 32, i8* %162
	%163 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 17
	store i8 91, i8* %163
	%164 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 18
	store i8 58, i8* %164
	%165 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 19
	store i8 44, i8* %165
	%166 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 20
	store i8 32, i8* %166
	%167 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 21
	store i8 91, i8* %167
	%168 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 22
	store i8 46, i8* %168
	%169 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 23
	store i8 44, i8* %169
	%170 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 24
	store i8 32, i8* %170
	%171 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 25
	store i8 111, i8* %171
	%172 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 26
	store i8 114, i8* %172
	%173 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 27
	store i8 32, i8* %173
	%174 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 28
	store i8 91, i8* %174
	%175 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 29
	store i8 61, i8* %175
	%176 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 30
	store i8 0, i8* %176
	%177 = getelementptr [31 x i8], [31 x i8]* @.str.46, i32 0, i32 0
	%178 = bitcast i8* %177 to i32*
	store i32* %178, i32** %17
	br label %179

179:
	%180 = load i32, i32* %16
	%181 = icmp eq i32 %180, 3
	%182 = zext i1 %181 to i32
	%183 = icmp ne i32 %182, 0
	br i1 %183, label %184, label %211

184:
	%185 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 0
	store i8 73, i8* %185
	%186 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 1
	store i8 110, i8* %186
	%187 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 2
	store i8 118, i8* %187
	%188 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 3
	store i8 97, i8* %188
	%189 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 4
	store i8 108, i8* %189
	%190 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 5
	store i8 105, i8* %190
	%191 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 6
	store i8 100, i8* %191
	%192 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 7
	store i8 32, i8* %192
	%193 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 8
	store i8 99, i8* %193
	%194 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 9
	store i8 111, i8* %194
	%195 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 10
	store i8 110, i8* %195
	%196 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 11
	store i8 116, i8* %196
	%197 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 12
	store i8 101, i8* %197
	%198 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 13
	store i8 110, i8* %198
	%199 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 14
	store i8 116, i8* %199
	%200 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 15
	store i8 32, i8* %200
	%201 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 16
	store i8 111, i8* %201
	%202 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 17
	store i8 102, i8* %202
	%203 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 18
	store i8 32, i8* %203
	%204 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 19
	store i8 92, i8* %204
	%205 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 20
	store i8 123, i8* %205
	%206 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 21
	store i8 92, i8* %206
	%207 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 22
	store i8 125, i8* %207
	%208 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 23
	store i8 0, i8* %208
	%209 = getelementptr [24 x i8], [24 x i8]* @.str.47, i32 0, i32 0
	%210 = bitcast i8* %209 to i32*
	store i32* %210, i32** %17
	br label %211

211:
	%212 = load i32, i32* %16
	%213 = icmp eq i32 %212, 4
	%214 = zext i1 %213 to i32
	%215 = icmp ne i32 %214, 0
	br i1 %215, label %216, label %246

216:
	%217 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 0
	store i8 82, i8* %217
	%218 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 1
	store i8 101, i8* %218
	%219 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 2
	store i8 103, i8* %219
	%220 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 3
	store i8 117, i8* %220
	%221 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 4
	store i8 108, i8* %221
	%222 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 5
	store i8 97, i8* %222
	%223 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 6
	store i8 114, i8* %223
	%224 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 7
	store i8 32, i8* %224
	%225 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 8
	store i8 101, i8* %225
	%226 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 9
	store i8 120, i8* %226
	%227 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 10
	store i8 112, i8* %227
	%228 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 11
	store i8 114, i8* %228
	%229 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 12
	store i8 101, i8* %229
	%230 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 13
	store i8 115, i8* %230
	%231 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 14
	store i8 115, i8* %231
	%232 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 15
	store i8 105, i8* %232
	%233 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 16
	store i8 111, i8* %233
	%234 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 17
	store i8 110, i8* %234
	%235 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 18
	store i8 32, i8* %235
	%236 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 19
	store i8 116, i8* %236
	%237 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 20
	store i8 111, i8* %237
	%238 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 21
	store i8 111, i8* %238
	%239 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 22
	store i8 32, i8* %239
	%240 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 23
	store i8 98, i8* %240
	%241 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 24
	store i8 105, i8* %241
	%242 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 25
	store i8 103, i8* %242
	%243 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 26
	store i8 0, i8* %243
	%244 = getelementptr [27 x i8], [27 x i8]* @.str.48, i32 0, i32 0
	%245 = bitcast i8* %244 to i32*
	store i32* %245, i32** %17
	br label %246

246:
	%247 = load i32, i32* %16
	%248 = icmp eq i32 %247, 5
	%249 = zext i1 %248 to i32
	%250 = icmp ne i32 %249, 0
	br i1 %250, label %251, label %291

251:
	%252 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 0
	store i8 73, i8* %252
	%253 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 1
	store i8 110, i8* %253
	%254 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 2
	store i8 118, i8* %254
	%255 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 3
	store i8 97, i8* %255
	%256 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 4
	store i8 108, i8* %256
	%257 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 5
	store i8 105, i8* %257
	%258 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 6
	store i8 100, i8* %258
	%259 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 7
	store i8 32, i8* %259
	%260 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 8
	store i8 112, i8* %260
	%261 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 9
	store i8 114, i8* %261
	%262 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 10
	store i8 101, i8* %262
	%263 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 11
	store i8 99, i8* %263
	%264 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 12
	store i8 101, i8* %264
	%265 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 13
	store i8 100, i8* %265
	%266 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 14
	store i8 105, i8* %266
	%267 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 15
	store i8 110, i8* %267
	%268 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 16
	store i8 103, i8* %268
	%269 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 17
	store i8 32, i8* %269
	%270 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 18
	store i8 114, i8* %270
	%271 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 19
	store i8 101, i8* %271
	%272 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 20
	store i8 103, i8* %272
	%273 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 21
	store i8 117, i8* %273
	%274 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 22
	store i8 108, i8* %274
	%275 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 23
	store i8 97, i8* %275
	%276 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 24
	store i8 114, i8* %276
	%277 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 25
	store i8 32, i8* %277
	%278 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 26
	store i8 101, i8* %278
	%279 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 27
	store i8 120, i8* %279
	%280 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 28
	store i8 112, i8* %280
	%281 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 29
	store i8 114, i8* %281
	%282 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 30
	store i8 101, i8* %282
	%283 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 31
	store i8 115, i8* %283
	%284 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 32
	store i8 115, i8* %284
	%285 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 33
	store i8 105, i8* %285
	%286 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 34
	store i8 111, i8* %286
	%287 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 35
	store i8 110, i8* %287
	%288 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 36
	store i8 0, i8* %288
	%289 = getelementptr [37 x i8], [37 x i8]* @.str.49, i32 0, i32 0
	%290 = bitcast i8* %289 to i32*
	store i32* %290, i32** %17
	br label %291

291:
	%292 = load i32, i32* %16
	%293 = icmp eq i32 %292, 6
	%294 = zext i1 %293 to i32
	%295 = icmp ne i32 %294, 0
	br i1 %295, label %296, label %318

296:
	%297 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 0
	store i8 84, i8* %297
	%298 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 1
	store i8 114, i8* %298
	%299 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 2
	store i8 97, i8* %299
	%300 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 3
	store i8 105, i8* %300
	%301 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 4
	store i8 108, i8* %301
	%302 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 5
	store i8 105, i8* %302
	%303 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 6
	store i8 110, i8* %303
	%304 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 7
	store i8 103, i8* %304
	%305 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 8
	store i8 32, i8* %305
	%306 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 9
	store i8 98, i8* %306
	%307 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 10
	store i8 97, i8* %307
	%308 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 11
	store i8 99, i8* %308
	%309 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 12
	store i8 107, i8* %309
	%310 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 13
	store i8 115, i8* %310
	%311 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 14
	store i8 108, i8* %311
	%312 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 15
	store i8 97, i8* %312
	%313 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 16
	store i8 115, i8* %313
	%314 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 17
	store i8 104, i8* %314
	%315 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 18
	store i8 0, i8* %315
	%316 = getelementptr [19 x i8], [19 x i8]* @.str.50, i32 0, i32 0
	%317 = bitcast i8* %316 to i32*
	store i32* %317, i32** %17
	br label %318

318:
	%319 = load i32, i32* %16
	%320 = icmp eq i32 %319, 7
	%321 = zext i1 %320 to i32
	%322 = icmp ne i32 %321, 0
	br i1 %322, label %323, label %344

323:
	%324 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 0
	store i8 73, i8* %324
	%325 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 1
	store i8 110, i8* %325
	%326 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 2
	store i8 118, i8* %326
	%327 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 3
	store i8 97, i8* %327
	%328 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 4
	store i8 108, i8* %328
	%329 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 5
	store i8 105, i8* %329
	%330 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 6
	store i8 100, i8* %330
	%331 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 7
	store i8 32, i8* %331
	%332 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 8
	store i8 114, i8* %332
	%333 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 9
	store i8 97, i8* %333
	%334 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 10
	store i8 110, i8* %334
	%335 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 11
	store i8 103, i8* %335
	%336 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 12
	store i8 101, i8* %336
	%337 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 13
	store i8 32, i8* %337
	%338 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 14
	store i8 101, i8* %338
	%339 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 15
	store i8 110, i8* %339
	%340 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 16
	store i8 100, i8* %340
	%341 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 17
	store i8 0, i8* %341
	%342 = getelementptr [18 x i8], [18 x i8]* @.str.51, i32 0, i32 0
	%343 = bitcast i8* %342 to i32*
	store i32* %343, i32** %17
	br label %344

344:
	%345 = load i32, i32* %16
	%346 = icmp eq i32 %345, 8
	%347 = zext i1 %346 to i32
	%348 = icmp ne i32 %347, 0
	br i1 %348, label %349, label %374

349:
	%350 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 0
	store i8 101, i8* %350
	%351 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 1
	store i8 109, i8* %351
	%352 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 2
	store i8 112, i8* %352
	%353 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 3
	store i8 116, i8* %353
	%354 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 4
	store i8 121, i8* %354
	%355 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 5
	store i8 32, i8* %355
	%356 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 6
	store i8 40, i8* %356
	%357 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 7
	store i8 115, i8* %357
	%358 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 8
	store i8 117, i8* %358
	%359 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 9
	store i8 98, i8* %359
	%360 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 10
	store i8 41, i8* %360
	%361 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 11
	store i8 101, i8* %361
	%362 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 12
	store i8 120, i8* %362
	%363 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 13
	store i8 112, i8* %363
	%364 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 14
	store i8 114, i8* %364
	%365 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 15
	store i8 101, i8* %365
	%366 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 16
	store i8 115, i8* %366
	%367 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 17
	store i8 115, i8* %367
	%368 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 18
	store i8 105, i8* %368
	%369 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 19
	store i8 111, i8* %369
	%370 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 20
	store i8 110, i8* %370
	%371 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 21
	store i8 0, i8* %371
	%372 = getelementptr [22 x i8], [22 x i8]* @.str.52, i32 0, i32 0
	%373 = bitcast i8* %372 to i32*
	store i32* %373, i32** %17
	br label %374

374:
	%375 = load i32, i32* %16
	%376 = icmp eq i32 %375, 9
	%377 = zext i1 %376 to i32
	%378 = icmp ne i32 %377, 0
	br i1 %378, label %379, label %411

379:
	%380 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 0
	store i8 73, i8* %380
	%381 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 1
	store i8 110, i8* %381
	%382 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 2
	store i8 118, i8* %382
	%383 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 3
	store i8 97, i8* %383
	%384 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 4
	store i8 108, i8* %384
	%385 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 5
	store i8 105, i8* %385
	%386 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 6
	store i8 100, i8* %386
	%387 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 7
	store i8 32, i8* %387
	%388 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 8
	store i8 99, i8* %388
	%389 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 9
	store i8 104, i8* %389
	%390 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 10
	store i8 97, i8* %390
	%391 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 11
	store i8 114, i8* %391
	%392 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 12
	store i8 97, i8* %392
	%393 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 13
	store i8 99, i8* %393
	%394 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 14
	store i8 116, i8* %394
	%395 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 15
	store i8 101, i8* %395
	%396 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 16
	store i8 114, i8* %396
	%397 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 17
	store i8 32, i8* %397
	%398 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 18
	store i8 99, i8* %398
	%399 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 19
	store i8 108, i8* %399
	%400 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 20
	store i8 97, i8* %400
	%401 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 21
	store i8 115, i8* %401
	%402 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 22
	store i8 115, i8* %402
	%403 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 23
	store i8 32, i8* %403
	%404 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 24
	store i8 110, i8* %404
	%405 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 25
	store i8 97, i8* %405
	%406 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 26
	store i8 109, i8* %406
	%407 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 27
	store i8 101, i8* %407
	%408 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 28
	store i8 0, i8* %408
	%409 = getelementptr [29 x i8], [29 x i8]* @.str.53, i32 0, i32 0
	%410 = bitcast i8* %409 to i32*
	store i32* %410, i32** %17
	br label %411

411:
	%412 = load i32*, i32** %17
	%413 = bitcast i32* %412 to i32*
	store i32* %413, i32** @re_err
	ret i32 -1

dead229:
	br label %106

414:
	%415 = load i32, i32* %13
	%416 = load i32, i32* %18
	%417 = icmp sle i32 %415, %416
	%418 = zext i1 %417 to i32
	%419 = icmp ne i32 %418, 0
	br i1 %419, label %420, label %421

420:
	store i32 0, i32* %12
	br label %422

421:
	ret i32 0

422:
	%423 = load i32, i32* %12
	%424 = icmp slt i32 %423, 192
	%425 = zext i1 %424 to i32
	%426 = icmp ne i32 %425, 0
	br i1 %426, label %427, label %433

427:
	%428 = load i32, i32* %12
	%429 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %428
	store i32 -1, i32* %429
	br label %430

430:
	%431 = load i32, i32* %12
	%432 = add i32 %431, 1
	store i32 %432, i32* %12
	br label %422

433:
	%434 = load i32, i32* %13
	%435 = call i32 @re_run(i32 0, i32 %434)
	store i32 %435, i32* %20
	%436 = load i32, i32* %20
	%437 = icmp eq i32 %436, -2
	%438 = zext i1 %437 to i32
	%439 = icmp ne i32 %438, 0
	br i1 %439, label %440, label %441

440:
	ret i32 -2

441:
	%442 = load i32, i32* %20
	%443 = icmp eq i32 %442, 1
	%444 = zext i1 %443 to i32
	%445 = icmp ne i32 %444, 0
	br i1 %445, label %446, label %454

dead230:
	br label %441

446:
	%447 = load i32, i32* %9
	%448 = sdiv i32 %447, 2
	store i32 %448, i32* %21
	%449 = load i32, i32* %19
	%450 = load i32, i32* %21
	%451 = icmp slt i32 %449, %450
	%452 = zext i1 %451 to i32
	%453 = icmp ne i32 %452, 0
	br i1 %453, label %457, label %459

454:
	%455 = load i32, i32* %13
	%456 = add i32 %455, 1
	store i32 %456, i32* %13
	br label %414

457:
	%458 = load i32, i32* %19
	br label %461

459:
	%460 = load i32, i32* %21
	br label %461

461:
	%462 = phi i32 [ %458, %457 ], [ %460, %459 ]
	store i32 %462, i32* %22
	%463 = load i32, i32* %14
	%464 = icmp ne i32 %463, 0
	br i1 %464, label %465, label %468

465:
	%466 = load i32, i32* %22
	%467 = mul i32 %466, 2
	br label %469

468:
	br label %469

469:
	%470 = phi i32 [ %467, %465 ], [ 0, %468 ]
	store i32 %470, i32* %23
	store i32 0, i32* %11
	br label %471

471:
	%472 = load i32, i32* %11
	%473 = load i32, i32* %23
	%474 = icmp slt i32 %472, %473
	%475 = zext i1 %474 to i32
	%476 = icmp ne i32 %475, 0
	br i1 %476, label %477, label %487

477:
	%478 = load i32, i32* %11
	%479 = load i32*, i32** %8
	%480 = getelementptr i32, i32* %479, i32 %478
	%481 = load i32, i32* %11
	%482 = getelementptr [192 x i32], [192 x i32]* @re_slot, i32 0, i32 %481
	%483 = load i32, i32* %482
	store i32 %483, i32* %480
	br label %484

484:
	%485 = load i32, i32* %11
	%486 = add i32 %485, 1
	store i32 %486, i32* %11
	br label %471

487:
	%488 = load i32, i32* %19
	ret i32 %488

dead231:
	br label %454

dead232:
	ret i32 0
}

define i32* @rt_regex_error() {
entry:
	%0 = alloca i32*
	%1 = load i32*, i32** @re_err
	%2 = bitcast i32* %1 to i32*
	store i32* %2, i32** %0
	%3 = load i32*, i32** %0
	%4 = bitcast i32* %3 to i32*
	%5 = inttoptr i32 0 to i32*
	%6 = icmp eq i32* %4, %5
	%7 = zext i1 %6 to i32
	%8 = icmp ne i32 %7, 0
	br i1 %8, label %9, label %12

9:
	%10 = getelementptr [1 x i8], [1 x i8]* @empty, i32 0, i32 0
	%11 = bitcast i8* %10 to i32*
	ret i32* %11

12:
	%13 = load i32*, i32** %0
	%14 = bitcast i32* %13 to i32*
	ret i32* %14

dead233:
	br label %12

dead234:
	ret i32* null
}

define i32 @__mec_body_main() {
entry:
	ret i32 0

dead235:
	ret i32 0
}

define i32 @__mec_ginit() {
entry:
	store i32 1, i32* @sink_err
	ret i32 0
}



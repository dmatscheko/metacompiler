/* See mod.kt. */
public class Main {
    public static void main(String[] args) {
        long s = 0;
        long i = 0;
        while (i < 40000) {
            s = s + i % 7;
            i = i + 1;
        }
        System.out.println(s);
    }
}

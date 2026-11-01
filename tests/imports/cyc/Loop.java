/* Cyclic inheritance across files fail-test (JLS 8.1.4, 7.2). Loop.java is the
 * program; javac's source path finds cyc.Peer under the directory package cyc
 * names, and Mid as Mid.java beside this file. Peer extends Mid, Mid implements
 * Loop, and Loop extends Peer.Inner, which depends on Peer: javac 24 refuses
 * the program with "cyclic inheritance involving cyc.Peer". This is a
 * SHOULD-FAIL entry, run without -i. **/
package cyc;

import cyc.Peer.Inner;

public interface Loop extends Peer.Inner {}

class Main {
    public static void main(String[] args) {
        System.out.println("must not run");
    }
}

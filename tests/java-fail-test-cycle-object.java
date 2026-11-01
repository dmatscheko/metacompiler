/* Redeclared java.lang.Object fail-test (JLS 4.3.2). Object is the one class
 * without a supertype, and every type has Object as one, so an Object that
 * names any supertype depends on itself. javac 24.0.2 (the file placed at
 * java/lang/Object.java, --patch-module java.base=.) refuses it with "cyclic
 * inheritance involving java.lang.Runnable", the first type named. Both halves
 * must refuse it with that message, before they look for a main method. This
 * is a SHOULD-FAIL entry. **/
package java.lang;

class Object implements Runnable, Cloneable {}

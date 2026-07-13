// A project namespace imported by tests/csharp-test-multifile.cs (via -i tests/imports).
// 'using geo;' maps to this file (geo.cs); its class registers like the main file's.
namespace geo
{
    class Vec
    {
        public int X;
        public int Y;

        public Vec(int x, int y)
        {
            this.X = x;
            this.Y = y;
        }

        public int Dot(Vec o)
        {
            return this.X * o.X + this.Y * o.Y;
        }

        public static Vec Scale(Vec v, int f)
        {
            return new Vec(v.X * f, v.Y * f);
        }
    }
}

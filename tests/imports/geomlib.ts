// A small geometry library imported by tests/typescript-test-multifile.ts (found via the
// -i include root: mec -i tests/imports ...). 'import { Vec, scale } from "./geomlib"'
// maps to this file (geomlib.ts): the leading './' is stripped and '.ts' appended. The
// file is parsed with the same grammar; its top-level class, function and const register
// like the main file's (flat, global registration). The 'export' keywords are erased at
// run time and the named-binding list on the import side is cosmetic.

export class Vec {
    x: number;
    y: number;

    constructor(x: number, y: number) {
        this.x = x;
        this.y = y;
    }

    dot(o: Vec): number {
        return this.x * o.x + this.y * o.y;
    }

    // References Vec itself, so the imported code must be able to resolve its own class.
    add(o: Vec): Vec {
        return new Vec(this.x + o.x, this.y + o.y);
    }
}

// A top-level function that builds a Vec: exercises an imported free function.
export function scale(v: Vec, f: number): Vec {
    return new Vec(v.x * f, v.y * f);
}

// A top-level const: exercises an imported value binding.
export const ORIGIN_TAG: string = "origin";

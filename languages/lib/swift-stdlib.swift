// swift-stdlib.swift -- the parts of Swift's standard library that are ordinary
// Swift declarations, compiled ahead of a program that names one of them (both
// swift grammars, swStdlibLoad). Each mirrors the stdlib's own declaration.

enum Result<Success, Failure: Error> {
    case success(Success)
    case failure(Failure)

    func get() throws -> Success {
        switch self {
        case .success(let v): return v
        case .failure(let e): throw e
        }
    }

    func map<NewSuccess>(_ transform: (Success) -> NewSuccess) -> Result<NewSuccess, Failure> {
        switch self {
        case .success(let v): return .success(transform(v))
        case .failure(let e): return .failure(e)
        }
    }

    func mapError<NewFailure>(_ transform: (Failure) -> NewFailure) -> Result<Success, NewFailure> {
        switch self {
        case .success(let v): return .success(v)
        case .failure(let e): return .failure(transform(e))
        }
    }

    func flatMap<NewSuccess>(_ transform: (Success) -> Result<NewSuccess, Failure>) -> Result<NewSuccess, Failure> {
        switch self {
        case .success(let v): return transform(v)
        case .failure(let e): return .failure(e)
        }
    }
}

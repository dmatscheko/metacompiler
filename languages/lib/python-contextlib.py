# python-contextlib.py -- the contextlib module, written in Python and compiled
# with a program that imports it (core.importFile in both python grammars), so
# one implementation serves all three engines.
#
# Every public name lives on the module object only (_mec_mod_contextlib);
# helpers carry the _mec_cl prefix. The behaviour is Lib/contextlib.py's, error
# texts included: contextmanager (also as a decorator), suppress, closing,
# nullcontext, ExitStack, ContextDecorator and AbstractContextManager. There is
# no traceback object, so __exit__'s third argument is always None.


class _mec_cl_ContextDecorator:
    def _recreate_cm(self):
        return self

    def __call__(self, func):
        def inner(*args, **kwds):
            with self._recreate_cm():
                return func(*args, **kwds)
        inner.__name__ = func.__name__
        inner.__qualname__ = getattr(func, "__qualname__", func.__name__)
        inner.__doc__ = func.__doc__
        inner.__wrapped__ = func
        return inner


class _mec_cl_AbstractContextManager:
    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        return None


# The object a @contextmanager function returns: the generator runs up to its
# yield on __enter__, and __exit__ resumes it - or throws the exception in at
# the yield, where the generator may swallow it.
class _mec_cl_GeneratorContextManager(_mec_cl_ContextDecorator):
    def __init__(self, func, args, kwds):
        self.gen = func(*args, **kwds)
        self.func, self.args, self.kwds = func, args, kwds

    def _recreate_cm(self):
        return self.__class__(self.func, self.args, self.kwds)

    def __enter__(self):
        try:
            return next(self.gen)
        except StopIteration:
            raise RuntimeError("generator didn't yield") from None

    def __exit__(self, typ, value, traceback):
        if typ is None:
            try:
                next(self.gen)
            except StopIteration:
                return False
            else:
                try:
                    raise RuntimeError("generator didn't stop")
                finally:
                    self.gen.close()
        if value is None:
            value = typ()
        try:
            self.gen.throw(value)
        except StopIteration as exc:
            return exc is not value
        except RuntimeError as exc:
            if exc is value:
                return False
            if isinstance(value, StopIteration) and exc.__cause__ is value:
                return False
            raise
        except BaseException as exc:
            if exc is not value:
                raise
            return False
        try:
            raise RuntimeError("generator didn't stop after throw()")
        finally:
            self.gen.close()


class _mec_cl_suppress(_mec_cl_AbstractContextManager):
    def __init__(self, *exceptions):
        self._exceptions = exceptions

    def __enter__(self):
        pass

    def __exit__(self, exctype, excinst, exctb):
        if exctype is None:
            return False
        return issubclass(exctype, self._exceptions)


class _mec_cl_closing(_mec_cl_AbstractContextManager):
    def __init__(self, thing):
        self.thing = thing

    def __enter__(self):
        return self.thing

    def __exit__(self, *exc_info):
        self.thing.close()


class _mec_cl_nullcontext(_mec_cl_AbstractContextManager):
    def __init__(self, enter_result=None):
        self.enter_result = enter_result

    def __enter__(self):
        return self.enter_result

    def __exit__(self, *excinfo):
        pass


# The callbacks run last-registered first; one that swallows the exception (a
# truthy __exit__) hides it from the ones after it, and one that raises
# replaces it, the replaced exception becoming the new one's __context__.
class _mec_cl_ExitStack(_mec_cl_AbstractContextManager):
    def __init__(self):
        self._exit_callbacks = []

    def pop_all(self):
        new_stack = type(self)()
        new_stack._exit_callbacks = self._exit_callbacks
        self._exit_callbacks = []
        return new_stack

    def push(self, exit):
        exit_method = getattr(type(exit), "__exit__", None)
        if exit_method is None:
            self._exit_callbacks.append(exit)
        else:
            self._exit_callbacks.append(_mec_cl_bound_exit(exit, exit_method))
        return exit

    def enter_context(self, cm):
        cls = type(cm)
        enter = getattr(cls, "__enter__", None)
        exit = getattr(cls, "__exit__", None)
        if enter is None or exit is None:
            raise TypeError("'" + cls.__module__ + "." + cls.__qualname__
                            + "' object does not support the context manager protocol")
        result = enter(cm)
        self._exit_callbacks.append(_mec_cl_bound_exit(cm, exit))
        return result

    def callback(self, callback, /, *args, **kwds):
        def _exit_wrapper(exc_type, exc, tb):
            callback(*args, **kwds)
        _exit_wrapper.__wrapped__ = callback
        self._exit_callbacks.append(_exit_wrapper)
        return callback

    def close(self):
        self.__exit__(None, None, None)

    def __enter__(self):
        return self

    def __exit__(self, *exc_details):
        exc = exc_details[1]
        received = exc is not None
        suppressed = False
        pending = False
        while self._exit_callbacks:
            cb = self._exit_callbacks.pop()
            try:
                if exc is None:
                    hit = cb(None, None, None)
                else:
                    hit = cb(type(exc), exc, None)
                if hit:
                    suppressed = True
                    pending = False
                    exc = None
            except BaseException as new_exc:
                if exc is not None and new_exc is not exc and new_exc.__context__ is None:
                    new_exc.__context__ = exc
                pending = True
                exc = new_exc
        if pending:
            raise exc
        return received and suppressed


def _mec_cl_bound_exit(cm, cm_exit):
    def _exit_wrapper(exc_type, exc, tb):
        return cm_exit(cm, exc_type, exc, tb)
    return _exit_wrapper


class _mec_cl_module:
    __all__ = ["AbstractContextManager", "ContextDecorator", "ExitStack", "closing",
               "contextmanager", "nullcontext", "suppress"]
    AbstractContextManager = _mec_cl_AbstractContextManager
    ContextDecorator = _mec_cl_ContextDecorator
    ExitStack = _mec_cl_ExitStack
    closing = _mec_cl_closing
    nullcontext = _mec_cl_nullcontext
    suppress = _mec_cl_suppress

    @staticmethod
    def contextmanager(func):
        def helper(*args, **kwds):
            return _mec_cl_GeneratorContextManager(func, args, kwds)
        helper.__name__ = func.__name__
        helper.__qualname__ = getattr(func, "__qualname__", func.__name__)
        helper.__doc__ = func.__doc__
        helper.__wrapped__ = func
        return helper

    def __repr__(self):
        return "<module 'contextlib'>"


# The classes answer to their public names, as CPython's do.
for _mec_cl_c, _mec_cl_n in [(_mec_cl_ContextDecorator, "ContextDecorator"),
                             (_mec_cl_AbstractContextManager, "AbstractContextManager"),
                             (_mec_cl_GeneratorContextManager, "_GeneratorContextManager"),
                             (_mec_cl_suppress, "suppress"), (_mec_cl_closing, "closing"),
                             (_mec_cl_nullcontext, "nullcontext"), (_mec_cl_ExitStack, "ExitStack")]:
    _mec_cl_c.__name__ = _mec_cl_n
    _mec_cl_c.__qualname__ = _mec_cl_n

_mec_mod_contextlib = type("module", (_mec_cl_module,), {})()
_mec_mod_contextlib.__name__ = "contextlib"

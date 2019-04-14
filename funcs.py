from typing import Callable, Iterable, Union

import builtins
import functools
import itertools


PartialChain = Callable[[Callable[[Iterable], Iterable]], Iterable]


def chainable(f: Callable):
    """
    Annotation allows f to be called as a map-like function or as a partial.
    """
    def wrapper(apply: Callable, *args) -> Union[Iterable, PartialChain]:
        if len(args) > 1:
            raise TypeError("{func}() takes 1 or 2 positional arguments but "
                            "{argc} were given".format(
                                func=f.__name__,
                                argc=len(args) + 1,
                            ))
        if len(args) == 1:
            return f(apply, args[0])
        return functools.partial(f, apply)
    # Don't use @functools.wraps in order to preserve this new signature
    wrapper.__name__ = f.__name__
    wrapper.__doc__ = f.__doc__
    wrapper.__qualname__ = f.__qualname__
    wrapper.__wrapper__ = f
    return wrapper


@chainable
def map(func: Callable, iterable: Iterable) -> Iterable:
    """
    map() can be used as a partial map or identically to the
    builtin map
    """
    return builtins.map(func, iterable)


@chainable
def flat_map(func: Callable, iterable: Iterable) -> Iterable:
    """
    flat_map() is like map() but flattens the output once after mapping
    """
    return itertools.chain.from_iterable(map(func, iterable))


@chainable
def filter(func: Callable, iterable: Iterable) -> Iterable:
    """
    filter() can be used as a partial filter or identically to the
    builtin filter
    """
    return builtins.filter(func, iterable)


@chainable
def reduce(func: Callable, iterable: Iterable) -> Iterable:
    """
    reduce() can be used as a partial reduce or identically to the
    functools reduce
    """
    return functools.reduce(func, iterable)


@chainable
def tee(func: Callable, iterable: Iterable) -> Iterable:
    """
    Calls func for every item in iterable, but returns the original item.
    Very useful for debugging func_chain's.

    Example:
    >>> from funcs import func_chain, map, tee
    >>>
    >>> func_chain(
    >>>     range(5),
    >>>     tee(print),
    >>>     map(str),
    >>> )
    >>> # Prints 0 1 2 3 4
    >>> # Returns ['0', '1', '2', '3', '4']
    """
    for item in iterable:
        func(item)
        yield item


def split(*funcs: Callable) -> PartialChain:
    """
    For every item in iterable and every func in funcs, call func(item) and
    join the results in tuples. Iterable is only run through once.

    Example:
    >>> from funcs import split
    >>>
    >>> f = split(str, int)
    >>> f([2.5])
    >>> [('2.5', 2)]
    """
    funcs = list(funcs)

    def wrapper(iterable: Iterable) -> Iterable:
        for item in iterable:
            yield tuple(map(lambda f: f(item), funcs))
    return wrapper


def func_chain(iterable: Iterable, *funcs: Iterable[PartialChain]) -> Iterable:
    """
    funcchain passes the iterable through a pipeline of partial functions

    Example:
    >>> from funcs import filter, map, funcchain
    >>>
    >>> list(funcchain(
    >>>     range(10),
    >>>     filter(lambda x: x > 5),
    >>>     filter(lambda x: x % 2 == 0),
    >>>     map(str),
    >>> ))
    >>> ['6', '8']
    """
    return functools.reduce(lambda a, b: b(a), funcs, iterable)

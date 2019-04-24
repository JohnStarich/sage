from functools import wraps
from typing import Callable


def lazy(f: Callable):
    name = f.__name__

    @wraps(f)
    def cache_get(self):
        if name in self.__dict__:
            return self.__dict__[name]
        value = f(self)
        self.__dict__[name] = value
        return value

    @wraps(f)
    def cache_del(self):
        if name in self.__dict__:
            del self.__dict__[name]

    return property(fget=cache_get, fdel=cache_del)

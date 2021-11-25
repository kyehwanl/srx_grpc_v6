# Process this file with autoconf to produce a configure script.

AC_PREREQ([2.63])
AC_INIT([srx_grpc], [1.0], [kyehwanl@nist.gov])
#AM_INIT_AUTOMAKE([-Wall -Werror foreign])
#AC_CONFIG_MACRO_DIR([m4])
#AC_PROG_MAKE_SET
#LT_PREREQ([2.2])
#LT_INIT([dlopen])
AC_MSG_CHECKING([host architecture information])
if test "$HOSTTYPE" = "x86_64-linux"; then
  AC_MSG_RESULT([64 bit])
  _arch="64"
else
  if test "$HOSTTYPE" = "x86_64"; then
    AC_MSG_RESULT([64 bit])
    _arch="64"
  else
    AC_MSG_RESULT([default])
    _arch=""
  fi
fi

CPU_ARCH=$_arch


if test "${libdir}" = "\${exec_prefix}/lib" ; then
  libdir="${libdir}$CPU_ARCH"
fi

#AC_CONFIG_HEADERS([config.h])
AC_CONFIG_FILES([Makefile])
AC_OUTPUT

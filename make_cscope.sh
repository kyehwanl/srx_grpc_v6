#!/bin/sh

#find ./ -name '*.[ch]' | ctags -R -L -
#find ./ -name '*.[ch]' > cscope.files

#find $PWD -name '*.[ch]' | ctags -R -L -
#find $PWD -name '*.[ch]' > cscope.files

#find $PWD -name '*.[ch]' | grep -v 'test_install' | grep -v 'tools' | ctags -R -L -
#find $PWD -name '*.[ch]' | grep -v "test_install" | grep -v 'tools' > cscope.files

find $PWD -name '*.[ch]' -o -name '*.go' | grep -v 'test_install' | grep -v 'tools' | grep -v 'tests' | grep -v 'ospfd' \
    | grep -v 'ripngd' | grep -v 'ospfclient' | grep -v 'ospf6d' | grep -v 'isisd' | grep -v 'ripd' | grep -v '_inst' \
    | grep -v 'babeld' | grep -v 'client/srx/' | grep -v 'extras' | grep -v '_inst' | grep -v 'NIST-BGP-SRx' | ctags -R -L -

find $PWD -name '*.[ch]' -o -name '*.go' | grep -v 'test_install' | grep -v 'tools' | grep -v 'tests' | grep -v 'ospfd' \
    | grep -v 'ripngd' | grep -v 'ospfclient' | grep -v 'ospf6d' | grep -v 'isisd' | grep -v 'ripd' | grep -v '_inst' \
    | grep -v 'babeld' | grep -v 'client/srx/' | grep -v 'extras' | grep -v '_inst' | grep -v 'NIST-BGP-SRx' > cscope.files
cscope -bvq


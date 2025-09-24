obj-m += continfo.o

all:
	# Ensure this is a tab, not spaces
	make -C /lib/modules/$(shell uname -r)/build M="$(PWD)" modules

clean:
	# Ensure this is a tab, not spaces
	make -C /lib/modules/$(shell uname -r)/build M="$(PWD)" clean
all:
	cc -o instrument-c *.c -rdynamic -finstrument-functions -ldl
	cc -o instrument-cpp *.cpp -rdynamic -finstrument-functions -ldl -lstdc++

clean:
	rm -f instrument-c instrument-cpp

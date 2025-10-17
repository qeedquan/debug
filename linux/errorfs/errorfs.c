#define FUSE_USE_VERSION 35

#define _GNU_SOURCE
#include <assert.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <ctype.h>
#include <unistd.h>
#include <errno.h>
#include <err.h>
#include <pthread.h>
#include <fuse.h>

#define nelem(x) (sizeof(x) / sizeof(x[0]))

enum {
	GETATTR,
	READLINK,
	MKNOD,
	MKDIR,
	UNLINK,
	SYMLINK,
	RENAME,
	LINK,
	CHMOD,
	CHOWN,
	TRUNCATE,
	OPEN,
	READ,
	WRITE,
	STATFS,
	FLUSH,
	RELEASE,
	FSYNC,
	SETXATTR,
	GETXATTR,
	LISTXATTR,
	REMOVEXATTR,
	OPENDIR,
	READDIR,
	RELEASEDIR,
	FSYNCDIR,
	ACCESS,
	CREATE,
	LOCK,
	UTIMENS,
	BMAP,
	IOCTL,
	POLL,
	WRITE_BUF,
	READ_BUF,
	FLOCK,
	FALLOCATE,
	COPY_FILE_RANGE,
	LSEEK,
	MAXOPS
};

typedef struct {
	int type;
	int val;
} Code;

typedef struct {
	size_t ci[MAXOPS];
	int pi[MAXOPS];

	size_t off;
} FD;

typedef struct {
	int action;
	int foreground;
	int debug;
	int wrap;

	ssize_t binsz;

	const char *mount;
	const char *conf[64];
	size_t nconf;
	size_t ncode;

	Code *code[MAXOPS];
	size_t codelen[MAXOPS];

	pthread_mutex_t glk;
	FD gfd;
} FS;

FS fs = {
    .action = 0,
    .debug = 0,
    .foreground = 1,
    .wrap = 0,
    .binsz = 1ULL << 24,
    .ncode = 4096,
    .glk = PTHREAD_MUTEX_INITIALIZER,
};

void *
xcalloc(size_t nmemb, size_t size)
{
	void *p;

	p = calloc(nmemb, size);
	if (!p)
		abort();
	return p;
}

int
getcode(struct fuse_file_info *fi, int op, int *r)
{
	FD *f;
	Code *c;
	size_t *ci, cn;
	int *pi;

	*r = 0;
	f = (fi) ? (FD *)fi->fh : NULL;
	if (!fi || !f)
		return (fs.action == 0) ? 's' : 'f';

	pi = &f->pi[op];
	ci = &f->ci[op];
	cn = fs.codelen[op];

	if (fs.wrap && cn > 0)
		*ci %= cn;

	if (*ci >= cn) {
		switch (fs.action) {
		case 0:
		case 1:
			return (fs.action == 0) ? 's' : 'f';

		default:
			assert(0);
		}
	}

	c = &fs.code[op][*ci];
	switch (c->type) {
	case 'r':
		*r = c->val;
		*ci += 1;
		return 'r';

	case 's':
	case 'f':
		*pi += 1;
		if (*pi >= c->val) {
			*pi = 0;
			*ci += 1;
		}
		return c->type;

	default:
		assert(0);
	}
}

int
getglbcode(int op, int *r)
{
	struct fuse_file_info fi;
	int c;

	pthread_mutex_lock(&fs.glk);
	fi.fh = (uintptr_t)&fs.gfd;
	c = getcode(&fi, op, r);
	pthread_mutex_unlock(&fs.glk);
	return c;
}

int
retcode(struct fuse_file_info *fi, int op, int err)
{
	int c, r;

	c = getcode(fi, op, &r);
	if (c == 's')
		return 0;
	if (c == 'f')
		return -err;
	return r;
}

int
glbretcode(int op, int err)
{
	int c, r;

	c = getglbcode(op, &r);
	if (c == 's')
		return 0;
	if (c == 'f')
		return -err;
	return r;
}

int
fsgetattr(const char *path, struct stat *st, struct fuse_file_info *fi)
{
	int c, r;

	c = getcode(fi, GETATTR, &r);
	if (c == 'r')
		return r;

	memset(st, 0, sizeof(*st));
	st->st_gid = getgid();
	st->st_uid = getuid();
	if (!strcmp(path, "/")) {
		st->st_mode = S_IFDIR | 0755;
		st->st_nlink = 2;
	} else if (!strcmp(path, "/error.bin")) {
		st->st_mode = S_IFREG | 0777;
		st->st_nlink = 1;
		st->st_size = fs.binsz;
	} else
		r = -ENOENT;

	return r;
}

int
fsreadlink(const char *path, char *buf, size_t size)
{
	int c, r, n;

	n = snprintf(buf, size, "%s", path);
	c = getglbcode(READLINK, &r);
	if (c == 'f')
		return -EINVAL;
	return n;
}

int
fsmknod(const char *, mode_t, dev_t)
{
	return glbretcode(MKNOD, EINVAL);
}

int
fsmkdir(const char *, mode_t)
{
	return glbretcode(MKDIR, EINVAL);
}

int
fsunlink(const char *)
{
	return glbretcode(UNLINK, EINVAL);
}

int
fsrmdir(const char *)
{
	return glbretcode(UNLINK, EINVAL);
}

int
fssymlink(const char *, const char *)
{
	return glbretcode(SYMLINK, EINVAL);
}

int
fsrename(const char *, const char *, unsigned int)
{
	return glbretcode(RENAME, EINVAL);
}

int
fslink(const char *, const char *)
{
	return glbretcode(LINK, EINVAL);
}

int
fschmod(const char *, mode_t, struct fuse_file_info *)
{
	return glbretcode(CHMOD, EINVAL);
}

int
fschown(const char *, uid_t, gid_t, struct fuse_file_info *)
{
	return glbretcode(CHOWN, EINVAL);
}

int
fstruncate(const char *, off_t, struct fuse_file_info *)
{
	return glbretcode(TRUNCATE, EINVAL);
}

int
fsopen(const char *path, struct fuse_file_info *fi)
{
	int c, r;

	fi->fh = (uintptr_t)calloc(1, sizeof(FD));
	if (!fi->fh)
		return -ENOMEM;

	c = getcode(fi, OPEN, &r);
	if (c == 'r')
		return c;

	if (!strcmp(path, "/error.bin"))
		return 0;

	return -ENOENT;
}

int
fsread(const char *path, char *buf, size_t size, off_t off, struct fuse_file_info *fi)
{
	FD *f;
	int c, r;

	memset(buf, 0, size);

	if (strcmp(path, "/error.bin"))
		return -EINVAL;

	f = (FD *)fi->fh;
	c = getcode(fi, READ, &r);
	if (c == 'f')
		return -EIO;
	if (c == 'r')
		return r;

	if (off > fs.binsz)
		return 0;

	f->off += size;

	return size;
}

int
fswrite(const char *path, const char *, size_t size, off_t, struct fuse_file_info *fi)
{
	int c, r;

	if (strcmp(path, "/error.bin"))
		return -EINVAL;

	c = getcode(fi, WRITE, &r);
	if (c == 'f')
		return -EINVAL;
	if (c == 'r')
		return r;

	return size;
}

int
fsstatfs(const char *, struct statvfs *)
{
	return glbretcode(STATFS, EINVAL);
}

int
fsflush(const char *, struct fuse_file_info *fi)
{
	return retcode(fi, FLUSH, EINVAL);
}

int
fsrelease(const char *, struct fuse_file_info *fi)
{
	return retcode(fi, RELEASE, EINVAL);
}

int
fsfsync(const char *, int, struct fuse_file_info *fi)
{
	return retcode(fi, FSYNC, EINVAL);
}

int
fssetxattr(const char *, const char *, const char *, size_t, int)
{
	return glbretcode(SETXATTR, EINVAL);
}

int
fsgetxattr(const char *, const char *, char *, size_t)
{
	return glbretcode(GETXATTR, EINVAL);
}

int
fslistxattr(const char *, char *, size_t)
{
	return glbretcode(LISTXATTR, EINVAL);
}

int
fsremovexattr(const char *, const char *)
{
	return glbretcode(REMOVEXATTR, EINVAL);
}

int
fsopendir(const char *path, struct fuse_file_info *fi)
{
	int c, r;

	fi->fh = (uintptr_t)calloc(1, sizeof(FD));
	if (!fi->fh)
		return -ENOMEM;

	c = getcode(fi, OPENDIR, &r);
	if (c == 'r')
		return r;

	if (!strcmp(path, "/"))
		return 0;
	return -ENOENT;
}

int
fsreaddir(const char *path, void *data, fuse_fill_dir_t filler, off_t, struct fuse_file_info *fi, enum fuse_readdir_flags)
{
	int c, r;

	c = getcode(fi, READDIR, &r);
	if (c == 'r')
		return r;

	if (strcmp(path, "/"))
		return -ENOENT;

	filler(data, ".", NULL, 0, 0);
	filler(data, "..", NULL, 0, 0);
	filler(data, "error.bin", NULL, 0, 0);
	return 0;
}

int
fsreleasedir(const char *, struct fuse_file_info *fi)
{
	int c, r;

	c = getcode(fi, RELEASEDIR, &r);
	if (fi)
		free((void *)fi->fh);
	if (c == 'r')
		return r;
	return 0;
}

int
fsfsyncdir(const char *, int, struct fuse_file_info *fi)
{
	return retcode(fi, FSYNCDIR, EINVAL);
}

int
fsaccess(const char *path, int)
{
	if (!strcmp(path, "/"))
		return 0;
	return glbretcode(ACCESS, EINVAL);
}

int
fscreate(const char *, mode_t, struct fuse_file_info *)
{
	return glbretcode(CREATE, EINVAL);
}

int
fslock(const char *, struct fuse_file_info *fi, int, struct flock *)
{
	return retcode(fi, LOCK, EINVAL);
}

int
fsutimens(const char *, const struct timespec[2], struct fuse_file_info *fi)
{
	return retcode(fi, UTIMENS, EINVAL);
}

int
fsbmap(const char *, size_t, uint64_t *)
{
	return glbretcode(BMAP, EINVAL);
}

int
fsioctl(const char *, unsigned int, void *, struct fuse_file_info *fi, unsigned int, void *)
{
	return retcode(fi, IOCTL, EINVAL);
}

int
fspoll(const char *, struct fuse_file_info *fi, struct fuse_pollhandle *, unsigned *)
{
	return retcode(fi, POLL, EINVAL);
}

int
fsflock(const char *, struct fuse_file_info *fi, int)
{
	return retcode(fi, FLOCK, EINVAL);
}

int
fsfallocate(const char *, int, off_t, off_t, struct fuse_file_info *fi)
{
	return retcode(fi, FALLOCATE, EINVAL);
}

ssize_t
fscopyfilerange(const char *, struct fuse_file_info *fi, off_t, const char *, struct fuse_file_info *, off_t, size_t, int)
{
	return retcode(fi, COPY_FILE_RANGE, EINVAL);
}

off_t
fslseek(const char *, off_t off, int whence, struct fuse_file_info *fi)
{
	FD *f;
	int c, r;

	f = (FD *)fi->fh;
	c = getcode(fi, LSEEK, &r);
	if (c == 'f')
		return -EINVAL;
	if (c == 'r')
		return c;

	if (whence == SEEK_CUR)
		off += f->off;
	else if (whence == SEEK_END)
		off = fs.binsz + off;

	if (off < 0 || off >= fs.binsz)
		return -EINVAL;

	f->off = off;
	return 0;
}

void
usage(void)
{
	fprintf(stderr, "usage: [options] <mountdir>\n");
	fprintf(stderr, "  -a  specify action (default: %d)\n", fs.action);
	fprintf(stderr, "  -c  add config file\n");
	fprintf(stderr, "  -d  enable debugging (default: %d)\n", fs.debug);
	fprintf(stderr, "  -f  run in foreground (default: %d)\n", fs.foreground);
	fprintf(stderr, "  -h  show this message\n");
	fprintf(stderr, "  -n  code queue size (default: %zu)\n", fs.ncode);
	fprintf(stderr, "  -w  wrap around code queue (default: %d)", fs.wrap);
	fprintf(stderr, "\n");
	fprintf(stderr, "action after code queue is drained:\n");
	fprintf(stderr, "  0  always return success\n");
	fprintf(stderr, "  1  always return failure\n");
	exit(2);
}

int
parseconf(FS *fs, const char *name)
{
	static const char *ops[] = {
	    "GETATTR",
	    "READLINK",
	    "MKNOD",
	    "MKDIR",
	    "UNLINK",
	    "SYMLINK",
	    "RENAME",
	    "LINK",
	    "CHMOD",
	    "CHOWN",
	    "TRUNCATE",
	    "OPEN",
	    "READ",
	    "WRITE",
	    "STATFS",
	    "FLUSH",
	    "RELEASE",
	    "FSYNC",
	    "SETXATTR",
	    "GETXATTR",
	    "LISTXATTR",
	    "REMOVEXATTR",
	    "OPENDIR",
	    "READDIR",
	    "RELEASEDIR",
	    "FSYNCDIR",
	    "ACCESS",
	    "CREATE",
	    "LOCK",
	    "UTIMENS",
	    "BMAP",
	    "IOCTL",
	    "POLL",
	    "WRITE_BUF",
	    "READ_BUF",
	    "FLOCK",
	    "FALLOCATE",
	    "COPY_FILE_RANGE",
	    "LSEEK",
	};

	Code *c;
	FILE *fp;
	char line[1024];
	char *saveptr;
	char *op, *val;
	size_t i;
	int r;

	r = 0;
	fp = fopen(name, "rb");
	if (!fp)
		goto error;

	while (fgets(line, sizeof(line), fp)) {
		op = strtok_r(line, " ", &saveptr);
		if (!op)
			continue;

		for (i = 0; i < nelem(ops); i++) {
			if (!strcasecmp(op, ops[i]))
				break;
		}
		if (i == nelem(ops))
			continue;

		while ((val = strtok_r(NULL, " ", &saveptr))) {
			if (fs->codelen[i] >= fs->ncode) {
				errno = ENOMEM;
				goto error;
			}
			c = &fs->code[i][fs->codelen[i]++];

			if (val[0] == '-' || isdigit(val[0])) {
				c->type = 'r';
				c->val = atoi(val);
			} else if (val[0] == 's' || val[0] == 'f') {
				c->type = val[0];
				c->val = 1;
				if (isdigit(val[1]))
					c->val = atoi(val + 1);
			} else {
				errno = EINVAL;
				goto error;
			}
		}
	}

	if (0) {
	error:
		r = -errno;
	}

	if (fp)
		fclose(fp);
	return r;
}

int
parseopt(void *data, const char *arg, int key, struct fuse_args *)
{
	char *ep;
	int *c;

	c = data;
	switch (key) {
	case FUSE_OPT_KEY_OPT:
		break;

	case FUSE_OPT_KEY_NONOPT:
		switch (*c) {
		case 'a':
			fs.action = atoi(arg);
			break;

		case 'c':
			if (fs.nconf >= nelem(fs.conf))
				errx(1, "Too much config files specified");
			fs.conf[fs.nconf++] = arg;
			break;

		case 'd':
			fs.debug = atoi(arg);
			break;

		case 'f':
			fs.foreground = atoi(arg);
			break;

		case 'h':
			usage();
			break;

		case 'n':
			fs.ncode = strtoul(arg, &ep, 0);
			break;

		case 'w':
			fs.wrap = atoi(arg);
			break;

		default:
			fs.mount = arg;
			break;
		}
		*c = 0;
		break;

	default:
		*c = key;
		break;
	}
	return 0;
}

int
main(int argc, char *argv[])
{
	static const struct fuse_operations ops = {
	    .getattr = fsgetattr,
	    .readlink = fsreadlink,
	    .mknod = fsmknod,
	    .mkdir = fsmkdir,
	    .unlink = fsunlink,
	    .symlink = fssymlink,
	    .rename = fsrename,
	    .link = fslink,
	    .chmod = fschmod,
	    .chown = fschown,
	    .truncate = fstruncate,
	    .open = fsopen,
	    .read = fsread,
	    .write = fswrite,
	    .statfs = fsstatfs,
	    .setxattr = fssetxattr,
	    .getxattr = fsgetxattr,
	    .listxattr = fslistxattr,
	    .removexattr = fsremovexattr,
	    .opendir = fsopendir,
	    .readdir = fsreaddir,
	    .releasedir = fsreleasedir,
	    .fsyncdir = fsfsyncdir,
	    .access = fsaccess,
	    .create = fscreate,
	    .lock = fslock,
	    .utimens = fsutimens,
	    .bmap = fsbmap,
	    .ioctl = fsioctl,
	    .poll = fspoll,
	    .write_buf = NULL,
	    .read_buf = NULL,
	    .flock = fsflock,
	    .fallocate = fsfallocate,
	    .copy_file_range = fscopyfilerange,
	    .lseek = fslseek,
	};

	static const struct fuse_opt opts[] = {
	    FUSE_OPT_KEY("-a", 'a'),
	    FUSE_OPT_KEY("-c", 'c'),
	    FUSE_OPT_KEY("-d", 'd'),
	    FUSE_OPT_KEY("-f", 'f'),
	    FUSE_OPT_KEY("-h", 'h'),
	    FUSE_OPT_KEY("-n", 'n'),
	    FUSE_OPT_KEY("-w", 'w'),
	    FUSE_OPT_END,
	};
	struct fuse_args args = FUSE_ARGS_INIT(argc, argv);

	size_t i;
	int c;
	int r;

	c = 0;
	if (fuse_opt_parse(&args, &c, opts, parseopt) != 0)
		errx(1, "Failed to parse options");

	if (fs.action < 0 || fs.action > 1)
		errx(1, "Invalid action specified");

	if (fs.ncode == 0)
		errx(1, "Invalid code queue size");

	if (!fs.mount)
		usage();

	for (i = 0; i < nelem(fs.code); i++)
		fs.code[i] = xcalloc(fs.ncode, sizeof(*fs.code));

	for (i = 0; i < fs.nconf; i++) {
		if (parseconf(&fs, fs.conf[i]) < 0)
			errx(1, "Failed to parse config %s: %s", fs.conf[i], strerror(errno));
	}

	if (fs.foreground)
		fuse_opt_add_arg(&args, "-f");
	if (fs.debug)
		fuse_opt_add_arg(&args, "-d");
	fuse_opt_add_arg(&args, fs.mount);

	r = fuse_main(args.argc, args.argv, &ops, NULL);

	fuse_opt_free_args(&args);
	return r;
}

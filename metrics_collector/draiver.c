// SPDX-License-Identifier: GPL-2.0
#include <linux/init.h>
#include <linux/module.h>
#include <linux/fs.h>
#include <linux/miscdevice.h>
#include <linux/random.h>
#include <linux/slab.h>
#include <linux/uaccess.h>

#define DEVICE_NAME "myrand"
#define MAX_CHUNK   (PAGE_SIZE)

MODULE_LICENSE("GPL");
MODULE_AUTHOR("OpenAI");
MODULE_DESCRIPTION("Virtual char device like /dev/urandom");
MODULE_VERSION("1.0");

/*
 * read():
 *  - генерирует count байт случайных данных
 *  - копирует их в userspace
 */
static ssize_t myrand_read(struct file *file, char __user *buf,
                           size_t count, loff_t *ppos)
{
    u8 *kbuf;
    size_t chunk;
    size_t done = 0;
    int ret;

    if (!count)
        return 0;

    while (done < count) {
        chunk = min_t(size_t, count - done, MAX_CHUNK);

        kbuf = kmalloc(chunk, GFP_KERNEL);
        if (!kbuf)
            return done ? (ssize_t)done : -ENOMEM;

        /*
         * Заполняем буфер случайными байтами.
         * Это даже лучше, чем просто "псевдослучайные" данные:
         * источник похож по идее на /dev/urandom.
         */
        get_random_bytes(kbuf, chunk);

        ret = copy_to_user(buf + done, kbuf, chunk);
        kfree(kbuf);

        if (ret) {
            size_t copied = chunk - ret;
            done += copied;
            return done ? (ssize_t)done : -EFAULT;
        }

        done += chunk;
    }

    return done;
}

/*
 * write():
 *  - принимает любые данные
 *  - ничего с ними не делает
 *  - сообщает, что всё "успешно записано"
 */
static ssize_t myrand_write(struct file *file, const char __user *buf,
                            size_t count, loff_t *ppos)
{
    return count;
}

static const struct file_operations myrand_fops = {
    .owner = THIS_MODULE,
    .read  = myrand_read,
    .write = myrand_write,
    .llseek = no_llseek,
};

static struct miscdevice myrand_dev = {
    .minor = MISC_DYNAMIC_MINOR,
    .name  = DEVICE_NAME,
    .fops  = &myrand_fops,
    .mode  = 0666, /* чтобы /dev/myrand был доступен всем */
};

static int __init myrand_init(void)
{
    int ret;

    ret = misc_register(&myrand_dev);
    if (ret) {
        pr_err("myrand: misc_register failed: %d\n", ret);
        return ret;
    }

    pr_info("myrand: registered /dev/%s\n", DEVICE_NAME);
    return 0;
}

static void __exit myrand_exit(void)
{
    misc_deregister(&myrand_dev);
    pr_info("myrand: unregistered /dev/%s\n", DEVICE_NAME);
}

module_init(myrand_init);
module_exit(myrand_exit);
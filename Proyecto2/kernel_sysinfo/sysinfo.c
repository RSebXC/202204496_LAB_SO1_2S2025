#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/sched.h>
#include <linux/mm.h>
#include <linux/seq_file.h>
#include <linux/jiffies.h> // Para medir tiempo

#define PROC_NAME "sysinfo_so1_202204496"

static unsigned long last_total_time = 0;
static unsigned long last_process_time = 0;

static int show_sysinfo(struct seq_file *m, void *v) {
    struct task_struct *p;
    unsigned long total_ram = totalram_pages() * PAGE_SIZE / 1024 / 1024; // Total RAM en MB
    unsigned long free_ram = si_mem_available() * PAGE_SIZE / 1024 / 1024; // RAM libre en MB
    unsigned long used_ram = total_ram - free_ram;

    seq_printf(m, "{\"total_ram_mb\": %lu, \"free_ram_mb\": %lu, \"used_ram_mb\": %lu, \"processes\": [", 
               total_ram, free_ram, used_ram);

    bool first = true;
    unsigned long total_jiffies = jiffies - INITIAL_JIFFIES; // Tiempo total del sistema
    for_each_process(p) {
        if (!first) seq_printf(m, ",");
        first = false;

        unsigned long vsz = p->mm ? (get_mm_rss(p->mm) * PAGE_SIZE / 1024) : 0; // VSZ en KB
        unsigned long rss = p->mm ? ((get_mm_counter(p->mm, MM_FILEPAGES) + 
                                    get_mm_counter(p->mm, MM_ANONPAGES)) * PAGE_SIZE / 1024) : 0; // RSS en KB
        long cpu_ticks = p->utime + p->stime; // Tiempo de CPU del proceso (jiffies)
        int mem_percent = total_ram > 0 ? (rss * 100) / (total_ram * 1024) : 0; // %Mem aproximado
        int cpu_percent = total_jiffies > 0 ? (cpu_ticks * 100) / total_jiffies : 0; // %CPU aproximado

        char state = task_state_to_char(p); // Estado del proceso

        seq_printf(m, "{\"pid\": %d, \"name\": \"%s\", \"cmdline\": \"%s\", \"vsz_kb\": %lu, \"rss_kb\": %lu, \"mem_percent\": %d, \"cpu_percent\": %d, \"state\": \"%c\"}", 
                   p->pid, p->comm, p->comm, vsz, rss, mem_percent, cpu_percent, state);
    }
    seq_printf(m, "]}\n");
    return 0;
}

static int sysinfo_open(struct inode *inode, struct file *file) {
    return single_open(file, show_sysinfo, NULL);
}

static const struct proc_ops sysinfo_fops = {
    .proc_open    = sysinfo_open,
    .proc_read    = seq_read,
    .proc_lseek   = seq_lseek,
    .proc_release = single_release,
};

static int __init sysinfo_init(void) {
    proc_create(PROC_NAME, 0, NULL, &sysinfo_fops);
    printk(KERN_INFO "Loaded sysinfo module for carnet 202204496\n");
    return 0;
}

static void __exit sysinfo_exit(void) {
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "Unloaded sysinfo module\n");
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Rodrigo Prime");
MODULE_DESCRIPTION("System info module");
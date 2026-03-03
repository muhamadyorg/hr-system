package goserver

import (
        "fmt"
        "log"

        "golang.org/x/crypto/bcrypt"
)

func SeedDatabase() error {
        existing, err := GetUserByUsername("sudo")
        if err != nil {
                return err
        }
        if existing != nil {
                log.Println("[Seed] Seed ma'lumotlari allaqachon mavjud, o'tkazildi")
                return nil
        }

        log.Println("[Seed] Boshlang'ich ma'lumotlar qo'shilmoqda...")

        sudoHash, _ := bcrypt.GenerateFromPassword([]byte("sudo123"), 10)
        sudoUser, err := CreateUser("sudo", string(sudoHash), "Super Admin", "sudo", nil, true)
        if err != nil {
                return err
        }
        log.Printf("[Seed] Sudo yaratildi: id=%d", sudoUser.ID)

        adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
        adminUser, err := CreateUser("admin", string(adminHash), "Admin Foydalanuvchi", "admin", &sudoUser.ID, true)
        if err != nil {
                return err
        }
        SetSetting("admin_password_"+itoa(adminUser.ID), "admin123")
        log.Printf("[Seed] Admin yaratildi: id=%d", adminUser.ID)

        groups := []struct {
                Name  string
                Login string
                Pass  string
                Desc  string
        }{
                {"IT", "it", "it123", "IT bo'limi"},
                {"Oshxona", "oshxona", "oshxona123", "Oshxona bo'limi"},
                {"Xavfsizlik", "xavfsizlik", "xavfsizlik123", "Xavfsizlik bo'limi"},
        }

        groupIDs := make([]int, 0, 3)
        for _, g := range groups {
                login := g.Login
                pass := g.Pass
                desc := g.Desc
                grp, err := CreateGroup(g.Name, &login, &pass, &desc, sudoUser.ID, nil, true)
                if err != nil {
                        log.Printf("[Seed] Guruh yaratishda xatolik %s: %v", g.Name, err)
                        continue
                }
                groupIDs = append(groupIDs, grp.ID)
                log.Printf("[Seed] Guruh yaratildi: %s (id=%d)", g.Name, grp.ID)
        }

        employees := []struct {
                No       string
                FullName string
                Position string
                GroupIdx int
                Phone    string
        }{
                {"EMP001", "Aliyev Sardor", "Senior Developer", 0, "+998901234567"},
                {"EMP002", "Karimova Dilnoza", "Frontend Developer", 0, "+998901234568"},
                {"EMP003", "Rahimov Jasur", "Backend Developer", 0, "+998901234569"},
                {"EMP004", "Toshmatov Bekzod", "Bosh oshpaz", 1, "+998901234570"},
                {"EMP005", "Yusupova Madina", "Oshpaz", 1, "+998901234571"},
                {"EMP006", "Nurmatov Otabek", "Qo'riqchi", 2, "+998901234572"},
                {"EMP007", "Abdullayev Sherzod", "Qo'riqchi", 2, "+998901234573"},
                {"EMP008", "Qodirov Ulugbek", "DevOps Engineer", 0, "+998901234574"},
        }

        for _, emp := range employees {
                pos := emp.Position
                phone := emp.Phone
                var gid *int
                if emp.GroupIdx < len(groupIDs) {
                        gid = &groupIDs[emp.GroupIdx]
                }
                _, err := CreateEmployee(emp.No, emp.FullName, &pos, gid, &phone, true, false)
                if err != nil {
                        log.Printf("[Seed] Xodim yaratishda xatolik %s: %v", emp.FullName, err)
                        continue
                }
        }

        log.Println("[Seed] Boshlang'ich ma'lumotlar muvaffaqiyatli qo'shildi")
        return nil
}

func itoa(i int) string {
        return fmt.Sprintf("%d", i)
}


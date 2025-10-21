# 📝 Message Template System Guide

## 🎯 Overview

The template system allows you to create multiple message variations and send them in **permutation order** to ensure each customer receives a unique message.

## ✨ Features

### **1. Auto-Discovery**
- Scans current directory for `.txt` and `.md` files
- Also checks `templates/` folder if it exists
- Loads all templates automatically

### **2. Interactive Selection**
- Preview all found templates
- Choose specific templates or use all
- Fall back to built-in defaults

### **3. Permutation Mode**
- Templates rotate automatically
- Each customer gets a different template
- Ensures message variety
- Prevents repetition

## 📁 File Structure

```
your-project/
├── main.go
├── customers.csv
├── template1.txt          ← Auto-discovered
├── template2.txt          ← Auto-discovered
├── template3.txt          ← Auto-discovered
└── templates/             ← Optional folder
    ├── welcome.txt
    ├── followup.md
    └── reminder.txt
```

## 📝 Creating Templates

### **Basic Template**
```
مرحباً {CustomerName}!

رقم العميل: {Code}

شكراً لك!
```

### **Available Placeholders**
- `{CustomerName}` - Customer's name
- `{Code}` - Customer code/ID
- `{Phone}` - Phone number
- `{Mobile}` - Mobile number

### **Template Examples**

#### **Template 1: Formal**
```
عزيزي {CustomerName}،

نود إعلامك بأن رقم حسابك هو: {Code}

نشكرك على ثقتك بنا.

مع تحياتنا،
فريق العمل
```

#### **Template 2: Friendly**
```
مرحباً {CustomerName}! 👋

كود العميل: {Code}

نتمنى لك يوماً سعيداً! ✨
```

#### **Template 3: Professional**
```
السيد/ة {CustomerName}

رقم العميل المسجل: {Code}
رقم الهاتف: {Mobile}

للاستفسارات، نحن في خدمتك.
```

## 🔄 How Permutation Works

### **Example with 3 Templates**

```
Customer 1 → Template 1
Customer 2 → Template 2
Customer 3 → Template 3
Customer 4 → Template 1  (rotation)
Customer 5 → Template 2
Customer 6 → Template 3
...and so on
```

### **Benefits**
- ✅ Each customer gets unique message
- ✅ Looks more natural (not automated)
- ✅ Reduces spam detection
- ✅ Better engagement rates

## 🚀 Usage Flow

### **1. Create Templates**
```bash
# Create template files
echo "مرحباً {CustomerName}!" > template1.txt
echo "عزيزي {CustomerName}" > template2.txt
echo "أهلاً {CustomerName}" > template3.txt
```

### **2. Run Application**
```bash
go run main.go
```

### **3. Select Templates**
```
📝 Message Templates Found
────────────────────────────────────────────────────────────
Found 3 template files

? Select templates to use:
  ▸ Template 1: مرحباً {CustomerName}!...
    Template 2: عزيزي {CustomerName}...
    Template 3: أهلاً {CustomerName}...
    ✓ Use ALL templates (Recommended)
    ⚙ Use built-in default templates
```

### **4. Automatic Rotation**
```
ℹ Template Configuration
  Using 3 message template(s) in permutation mode

  • Each customer will receive a different template
  • Templates rotate automatically
  • Ensures message variety
```

## 📊 Template Statistics

During execution, you'll see which template was used:

```
[2025-01-21 14:30:15] SUCCESS  Message sent to Ahmed (Template 1/3)
[2025-01-21 14:30:22] SUCCESS  Message sent to Sara (Template 2/3)
[2025-01-21 14:30:29] SUCCESS  Message sent to Omar (Template 3/3)
[2025-01-21 14:30:36] SUCCESS  Message sent to Fatima (Template 1/3)
```

## 🎨 Advanced Templates

### **Multi-Line with Formatting**
```
╔══════════════════════════════════╗
║   مرحباً {CustomerName}!         ║
╚══════════════════════════════════╝

📋 معلومات الحساب:
   • الكود: {Code}
   • الهاتف: {Mobile}

شكراً لتعاملك معنا! 🌟
```

### **With Emojis**
```
👋 مرحباً {CustomerName}!

🎉 نحن سعداء بخدمتك
📱 رقمك: {Code}
✨ نتطلع لخدمتك دائماً

💙 فريق العمل
```

### **Markdown Format** (template.md)
```markdown
# مرحباً {CustomerName}!

## معلومات العميل
- **الكود**: {Code}
- **الهاتف**: {Mobile}

---

شكراً لثقتك بنا! ✨
```

## 🛠️ Best Practices

### **1. Template Variety**
- Create 3-5 different templates
- Vary the tone (formal/casual)
- Use different greetings
- Mix Arabic/English if appropriate

### **2. Personalization**
- Always use `{CustomerName}`
- Include `{Code}` for reference
- Keep it relevant and concise

### **3. Length**
- Keep templates under 500 characters
- WhatsApp has message limits
- Shorter messages get better engagement

### **4. Testing**
- Test all templates first
- Check placeholder replacement
- Verify formatting on mobile

## 🔍 Troubleshooting

### **No Templates Found**
```
 ⚠ WARNING 
┌─ No Templates Found
│  No .txt or .md template files found in current directory
│
├─ Tips:
│  • Using built-in default templates
│  • Create .txt or .md files with your message templates
│  • Place them in current directory or templates/ folder
└──────────────────────────────────────────────────────────
```

**Solution**: Create `.txt` or `.md` files in the same directory as `main.go`

### **Template Not Loading**
- Check file encoding (use UTF-8)
- Ensure file has content
- Check file permissions
- Verify file extension (.txt or .md)

### **Placeholders Not Replaced**
- Use exact placeholder names: `{CustomerName}`, `{Code}`, `{Phone}`, `{Mobile}`
- Case-sensitive
- No spaces inside braces

## 📈 Performance

### **Template Loading**
- Instant for small files (<100KB)
- Cached in memory
- No performance impact

### **Permutation**
- O(1) complexity
- No memory overhead
- Efficient rotation

## 💡 Pro Tips

1. **Seasonal Templates**: Create different templates for holidays/seasons
2. **A/B Testing**: Use different templates to test engagement
3. **Backup**: Keep template files in version control
4. **Organize**: Use `templates/` folder for many templates
5. **Preview**: Always preview before sending to all customers

## 📚 Example Template Set

### **Set 1: Customer Appreciation**
```
template_thanks1.txt
template_thanks2.txt
template_thanks3.txt
```

### **Set 2: Reminders**
```
template_reminder1.txt
template_reminder2.txt
```

### **Set 3: Promotions**
```
template_promo1.txt
template_promo2.txt
template_promo3.txt
```

## 🎯 Summary

- ✅ **Easy**: Just create `.txt` or `.md` files
- ✅ **Flexible**: Use any number of templates
- ✅ **Smart**: Automatic permutation
- ✅ **Professional**: Variety prevents spam detection
- ✅ **Efficient**: No performance overhead

---

**Happy Messaging!** 🚀📱

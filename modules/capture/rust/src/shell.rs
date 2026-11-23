use std::collections::HashMap;
use std::ffi::c_void;
use std::path::{Path, PathBuf};
use std::ptr::null_mut;
use std::sync::{Mutex, OnceLock};
use windows::core::{w, HSTRING};
use windows::Win32::Graphics::Gdi::{
    DeleteObject, GetDC, GetDIBits, GetObjectW, ReleaseDC, BITMAP, BITMAPINFO, BITMAPINFOHEADER, DIB_RGB_COLORS, HBITMAP,
};
use windows::Win32::Storage::FileSystem::{
    GetFileVersionInfoSizeW, GetFileVersionInfoW, VerQueryValueW, FILE_FLAGS_AND_ATTRIBUTES,
};
use windows::Win32::UI::Shell::{SHGetFileInfoW, SHFILEINFOW, SHGFI_ICON, SHGFI_LARGEICON};
use windows::Win32::UI::WindowsAndMessaging::{DestroyIcon, GetIconInfo, HICON, ICONINFO};

/// What Explorer shows a program as.
#[derive(Clone, Default)]
pub struct Look {
    /// The description in the program's version information, or "".
    pub name: String,
    /// The program's icon as a PNG, or empty.
    pub icon: Vec<u8>,
}

/// The look of the program at path, read once for each program.
pub fn look(path: &Path) -> Look {
    static SEEN: OnceLock<Mutex<HashMap<PathBuf, Look>>> = OnceLock::new();
    let seen = SEEN.get_or_init(Default::default);
    if let Some(look) = seen.lock().unwrap().get(path) {
        return look.clone();
    }
    let look = Look { name: description(path).unwrap_or_default(), icon: icon(path).unwrap_or_default() };
    seen.lock().unwrap().insert(path.to_path_buf(), look.clone());
    look
}

fn description(path: &Path) -> Option<String> {
    let file = HSTRING::from(path.as_os_str());
    unsafe {
        let size = GetFileVersionInfoSizeW(&file, None);
        if size == 0 {
            return None;
        }
        let mut data = vec![0u8; size as usize];
        GetFileVersionInfoW(&file, None, size, data.as_mut_ptr().cast()).ok()?;
        let block: *const c_void = data.as_ptr().cast();

        let mut languages: *mut c_void = null_mut();
        let mut length = 0u32;
        if !VerQueryValueW(block, w!("\\VarFileInfo\\Translation"), &mut languages, &mut length).as_bool() || length < 4 {
            return None;
        }
        let [language, codepage] = *languages.cast::<[u16; 2]>();
        let key = HSTRING::from(format!("\\StringFileInfo\\{language:04x}{codepage:04x}\\FileDescription"));
        let mut text: *mut c_void = null_mut();
        if !VerQueryValueW(block, &key, &mut text, &mut length).as_bool() || length == 0 {
            return None;
        }
        let chars = std::slice::from_raw_parts(text.cast::<u16>(), length as usize);
        let name = String::from_utf16_lossy(chars).trim_end_matches('\0').trim().to_string();
        (!name.is_empty()).then_some(name)
    }
}

fn icon(path: &Path) -> Option<Vec<u8>> {
    unsafe {
        let mut info = SHFILEINFOW::default();
        let found = SHGetFileInfoW(
            &HSTRING::from(path.as_os_str()),
            FILE_FLAGS_AND_ATTRIBUTES(0),
            Some(&mut info),
            size_of::<SHFILEINFOW>() as u32,
            SHGFI_ICON | SHGFI_LARGEICON,
        );
        if found == 0 || info.hIcon.is_invalid() {
            return None;
        }
        let image = pixels(info.hIcon);
        let _ = DestroyIcon(info.hIcon);
        let (width, height, rgba) = image?;
        Some(png(width, height, &rgba))
    }
}

/// The picture of an icon as RGBA, top row first.
unsafe fn pixels(icon: HICON) -> Option<(u32, u32, Vec<u8>)> {
    let mut parts = ICONINFO::default();
    GetIconInfo(icon, &mut parts).ok()?;
    let read = read(parts.hbmColor);
    let _ = DeleteObject(parts.hbmColor.into());
    let _ = DeleteObject(parts.hbmMask.into());
    read
}

unsafe fn read(color: HBITMAP) -> Option<(u32, u32, Vec<u8>)> {
    if color.is_invalid() {
        return None;
    }
    let mut bitmap = BITMAP::default();
    if GetObjectW(color.into(), size_of::<BITMAP>() as i32, Some((&mut bitmap as *mut BITMAP).cast())) == 0 {
        return None;
    }
    let (width, height) = (bitmap.bmWidth as u32, bitmap.bmHeight as u32);
    if width == 0 || height == 0 || width > 512 || height > 512 {
        return None;
    }
    let mut info = BITMAPINFO {
        bmiHeader: BITMAPINFOHEADER {
            biSize: size_of::<BITMAPINFOHEADER>() as u32,
            biWidth: width as i32,
            biHeight: -(height as i32),
            biPlanes: 1,
            biBitCount: 32,
            ..Default::default()
        },
        ..Default::default()
    };
    let mut data = vec![0u8; (width * height * 4) as usize];
    let screen = GetDC(None);
    let rows = GetDIBits(screen, color, 0, height, Some(data.as_mut_ptr().cast()), &mut info, DIB_RGB_COLORS);
    ReleaseDC(None, screen);
    if rows == 0 {
        return None;
    }
    let opaque = data.chunks_exact(4).all(|pixel| pixel[3] == 0);
    for pixel in data.chunks_exact_mut(4) {
        pixel.swap(0, 2);
        if opaque {
            pixel[3] = 255;
        }
    }
    Some((width, height, data))
}

/// An RGBA picture as a PNG, its data stored as it is.
fn png(width: u32, height: u32, rgba: &[u8]) -> Vec<u8> {
    let mut raw = Vec::with_capacity(rgba.len() + height as usize);
    for row in rgba.chunks_exact(width as usize * 4) {
        raw.push(0);
        raw.extend_from_slice(row);
    }
    let mut packed = vec![0x78, 0x01];
    let mut blocks = raw.chunks(0xffff).peekable();
    while let Some(block) = blocks.next() {
        packed.push(blocks.peek().is_none() as u8);
        packed.extend_from_slice(&(block.len() as u16).to_le_bytes());
        packed.extend_from_slice(&(!(block.len() as u16)).to_le_bytes());
        packed.extend_from_slice(block);
    }
    let (mut a, mut b) = (1u32, 0u32);
    for byte in &raw {
        a = (a + *byte as u32) % 65521;
        b = (b + a) % 65521;
    }
    packed.extend_from_slice(&((b << 16) | a).to_be_bytes());

    let mut header = Vec::with_capacity(13);
    header.extend_from_slice(&width.to_be_bytes());
    header.extend_from_slice(&height.to_be_bytes());
    header.extend_from_slice(&[8, 6, 0, 0, 0]);

    let mut file = vec![0x89, b'P', b'N', b'G', 0x0d, 0x0a, 0x1a, 0x0a];
    for (kind, data) in [(b"IHDR", header.as_slice()), (b"IDAT", packed.as_slice()), (b"IEND", &[][..])] {
        file.extend_from_slice(&(data.len() as u32).to_be_bytes());
        let begin = file.len();
        file.extend_from_slice(kind);
        file.extend_from_slice(data);
        let sum = crc(&file[begin..]);
        file.extend_from_slice(&sum.to_be_bytes());
    }
    file
}

fn crc(data: &[u8]) -> u32 {
    let mut sum = !0u32;
    for byte in data {
        sum ^= *byte as u32;
        for _ in 0..8 {
            sum = if sum & 1 == 1 { (sum >> 1) ^ 0xedb8_8320 } else { sum >> 1 };
        }
    }
    !sum
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_picture_becomes_a_png() {
        let file = png(2, 1, &[255, 0, 0, 255, 0, 0, 255, 128]);
        assert_eq!(&file[..8], &[0x89, b'P', b'N', b'G', 0x0d, 0x0a, 0x1a, 0x0a]);
        assert_eq!(&file[12..16], b"IHDR");
        assert_eq!(crc(b"IEND"), 0xae42_6082);
        assert_eq!(&file[file.len() - 4..], &0xae42_6082u32.to_be_bytes());
    }

    #[test]
    fn explorer_has_a_name_and_an_icon() {
        let look = look(Path::new("C:\\Windows\\explorer.exe"));
        println!("explorer.exe: {:?}, an icon of {} bytes", look.name, look.icon.len());
        assert!(!look.name.is_empty());
        assert!(look.icon.len() > 1000);
    }
}

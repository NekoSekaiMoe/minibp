// Helper.java - Helper class
package com.example;

import java.util.*;
import java.util.function.Predicate;

public class Helper {
    public static void log(String message) {
        System.out.println("[LOG] " + message);
    }
    
    public static void logWithTime(String message) {
        Date now = new Date();
        System.out.println("[LOG] [" + now + "] " + message);
    }
    
    public static void logFormat(String format, Object... args) {
        System.out.println("[LOG] " + String.format(format, args));
    }
    
    public static boolean isEmpty(String str) {
        return str == null || str.isEmpty();
    }
    
    public static boolean isBlank(String str) {
        return str == null || str.trim().isEmpty();
    }
    
    public static String trim(String str) {
        return str == null ? "" : str.trim();
    }
    
    public static String capitalize(String str) {
        if (isEmpty(str)) return str;
        return Character.toUpperCase(str.charAt(0)) + str.substring(1);
    }
    
    public static String toUpperCase(String str) {
        return str == null ? "" : str.toUpperCase();
    }
    
    public static String toLowerCase(String str) {
        return str == null ? "" : str.toLowerCase();
    }
    
    public static boolean contains(String str, String sub) {
        return str != null && sub != null && str.contains(sub);
    }
    
    public static boolean startsWith(String str, String prefix) {
        return str != null && prefix != null && str.startsWith(prefix);
    }
    
    public static boolean endsWith(String str, String suffix) {
        return str != null && suffix != null && str.endsWith(suffix);
    }
    
    public static String substringBefore(String str, String delimiter) {
        if (str == null || delimiter == null) return str;
        int idx = str.indexOf(delimiter);
        return idx >= 0 ? str.substring(0, idx) : str;
    }
    
    public static String substringAfter(String str, String delimiter) {
        if (str == null || delimiter == null) return str;
        int idx = str.indexOf(delimiter);
        return idx >= 0 ? str.substring(idx + delimiter.length()) : str;
    }
    
    public static String join(String delimiter, List<String> list) {
        if (list == null || list.isEmpty()) return "";
        return String.join(delimiter, list);
    }
    
    public static List<String> split(String str, String delimiter) {
        List<String> result = new ArrayList<>();
        if (str == null || delimiter == null) return result;
        
        int start = 0;
        int end = str.indexOf(delimiter);
        while (end >= 0) {
            result.add(str.substring(start, end));
            start = end + delimiter.length();
            end = str.indexOf(delimiter, start);
        }
        result.add(str.substring(start));
        return result;
    }
    
    public static String repeat(String str, int count) {
        if (str == null || count <= 0) return "";
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < count; i++) {
            sb.append(str);
        }
        return sb.toString();
    }
    
    public static String reverse(String str) {
        if (str == null) return null;
        return new StringBuilder(str).reverse().toString();
    }
    
    public static boolean isNumeric(String str) {
        if (isEmpty(str)) return false;
        try {
            Double.parseDouble(str);
            return true;
        } catch (NumberFormatException e) {
            return false;
        }
    }
    
    public static boolean isAlpha(String str) {
        if (isEmpty(str)) return false;
        for (char c : str.toCharArray()) {
            if (!Character.isLetter(c)) return false;
        }
        return true;
    }
    
    public static boolean isAlphanumeric(String str) {
        if (isEmpty(str)) return false;
        for (char c : str.toCharArray()) {
            if (!Character.isLetterOrDigit(c)) return false;
        }
        return true;
    }
    
    public static int countWords(String str) {
        if (isBlank(str)) return 0;
        String trimmed = str.trim();
        int count = 0;
        int i = 0;
        while (i < trimmed.length()) {
            while (i < trimmed.length() && Character.isWhitespace(trimmed.charAt(i))) {
                i++;
            }
            if (i < trimmed.length()) {
                count++;
                while (i < trimmed.length() && !Character.isWhitespace(trimmed.charAt(i))) {
                    i++;
                }
            }
        }
        return count;
    }
    
    public static String leftPad(String str, int width, char pad) {
        if (str == null) str = "";
        if (str.length() >= width) return str;
        StringBuilder sb = new StringBuilder();
        for (int i = str.length(); i < width; i++) {
            sb.append(pad);
        }
        return sb + str;
    }
    
    public static String rightPad(String str, int width, char pad) {
        if (str == null) str = "";
        if (str.length() >= width) return str;
        StringBuilder sb = new StringBuilder(str);
        for (int i = str.length(); i < width; i++) {
            sb.append(pad);
        }
        return sb.toString();
    }
    
    public static String abbreviate(String str, int maxWidth) {
        if (str == null) return null;
        if (str.length() <= maxWidth) return str;
        return str.substring(0, maxWidth - 3) + "...";
    }
    
    public static <T> void swap(T[] arr, int i, int j) {
        if (arr == null || i < 0 || j < 0 || i >= arr.length || j >= arr.length) return;
        T temp = arr[i];
        arr[i] = arr[j];
        arr[j] = temp;
    }
    
    public static <T> void reverse(List<T> list) {
        if (list == null || list.size() <= 1) return;
        Collections.reverse(list);
    }
    
    public static <T> List<T> filter(List<T> list, Predicate<T> predicate) {
        List<T> result = new ArrayList<>();
        if (list == null || predicate == null) return result;
        for (T item : list) {
            if (predicate.test(item)) {
                result.add(item);
            }
        }
        return result;
    }
}
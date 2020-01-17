// Copyright 2020 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ----------------

/*
This test program is typically run indirectly, by the "wuffs test" or "wuffs
bench" commands. These commands take an optional "-mimic" flag to check that
Wuffs' output mimics (i.e. exactly matches) other libraries' output, such as
giflib for GIF, libpng for PNG, etc.

To manually run this test:

for CC in clang gcc; do
  $CC -std=c99 -Wall -Werror wbmp.c && ./a.out
  rm -f a.out
done

Each edition should print "PASS", amongst other information, and exit(0).

Add the "wuffs mimic cflags" (everything after the colon below) to the C
compiler flags (after the .c file) to run the mimic tests.

To manually run the benchmarks, replace "-Wall -Werror" with "-O3" and replace
the first "./a.out" with "./a.out -bench". Combine these changes with the
"wuffs mimic cflags" to run the mimic benchmarks.
*/

// !! wuffs mimic cflags: -DWUFFS_MIMIC

// Wuffs ships as a "single file C library" or "header file library" as per
// https://github.com/nothings/stb/blob/master/docs/stb_howto.txt
//
// To use that single file as a "foo.c"-like implementation, instead of a
// "foo.h"-like header, #define WUFFS_IMPLEMENTATION before #include'ing or
// compiling it.
#define WUFFS_IMPLEMENTATION

// Defining the WUFFS_CONFIG__MODULE* macros are optional, but it lets users of
// release/c/etc.c whitelist which parts of Wuffs to build. That file contains
// the entire Wuffs standard library, implementing a variety of codecs and file
// formats. Without this macro definition, an optimizing compiler or linker may
// very well discard Wuffs code for unused codecs, but listing the Wuffs
// modules we use makes that process explicit. Preprocessing means that such
// code simply isn't compiled.
#define WUFFS_CONFIG__MODULES
#define WUFFS_CONFIG__MODULE__BASE
#define WUFFS_CONFIG__MODULE__WBMP

// If building this program in an environment that doesn't easily accommodate
// relative includes, you can use the script/inline-c-relative-includes.go
// program to generate a stand-alone C file.
#include "../../../release/c/wuffs-unsupported-snapshot.c"
#include "../testlib/testlib.c"
#ifdef WUFFS_MIMIC
// No mimic library.
#endif

// ---------------- WBMP Tests

const char* test_wuffs_wbmp_decode_interface() {
  CHECK_FOCUS(__func__);
  wuffs_wbmp__decoder dec;
  CHECK_STATUS("initialize",
               wuffs_wbmp__decoder__initialize(
                   &dec, sizeof dec, WUFFS_VERSION,
                   WUFFS_INITIALIZE__LEAVE_INTERNAL_BUFFERS_UNINITIALIZED));
  return do_test__wuffs_base__image_decoder(
      wuffs_wbmp__decoder__upcast_as__wuffs_base__image_decoder(&dec),
      "test/data/muybridge-frame-000.wbmp", 0, SIZE_MAX, 30, 20, 0xFFFFFFFF);
}

const char* test_wuffs_wbmp_decode_frame_config() {
  CHECK_FOCUS(__func__);
  wuffs_wbmp__decoder dec;
  CHECK_STATUS("initialize",
               wuffs_wbmp__decoder__initialize(
                   &dec, sizeof dec, WUFFS_VERSION,
                   WUFFS_INITIALIZE__LEAVE_INTERNAL_BUFFERS_UNINITIALIZED));

  wuffs_base__frame_config fc = ((wuffs_base__frame_config){});
  wuffs_base__io_buffer src = ((wuffs_base__io_buffer){
      .data = global_src_slice,
  });
  CHECK_STRING(read_file(&src, "test/data/hat.wbmp"));
  CHECK_STATUS("decode_frame_config #0",
               wuffs_wbmp__decoder__decode_frame_config(&dec, &fc, &src));

  wuffs_base__status status =
      wuffs_wbmp__decoder__decode_frame_config(&dec, &fc, &src);
  if (status.repr != wuffs_base__note__end_of_data) {
    RETURN_FAIL("decode_frame_config #1: got \"%s\", want \"%s\"", status.repr,
                wuffs_base__note__end_of_data);
  }
  if (src.meta.ri != src.meta.wi) {
    RETURN_FAIL("at end of data: ri (%zu) doesn't equal wi (%zu)", src.meta.ri,
                src.meta.wi);
  }
  return NULL;
}

const char* test_wuffs_wbmp_decode_image_config() {
  CHECK_FOCUS(__func__);
  wuffs_wbmp__decoder dec;
  CHECK_STATUS("initialize",
               wuffs_wbmp__decoder__initialize(
                   &dec, sizeof dec, WUFFS_VERSION,
                   WUFFS_INITIALIZE__LEAVE_INTERNAL_BUFFERS_UNINITIALIZED));

  wuffs_base__image_config ic = ((wuffs_base__image_config){});
  wuffs_base__io_buffer src = ((wuffs_base__io_buffer){
      .data = global_src_slice,
  });
  CHECK_STRING(read_file(&src, "test/data/bricks-nodither.wbmp"));
  CHECK_STATUS("decode_image_config",
               wuffs_wbmp__decoder__decode_image_config(&dec, &ic, &src));

  uint32_t got_width = wuffs_base__pixel_config__width(&ic.pixcfg);
  uint32_t want_width = 160;
  if (got_width != want_width) {
    RETURN_FAIL("width: got %" PRIu32 ", want %" PRIu32, got_width, want_width);
  }
  uint32_t got_height = wuffs_base__pixel_config__height(&ic.pixcfg);
  uint32_t want_height = 120;
  if (got_height != want_height) {
    RETURN_FAIL("height: got %" PRIu32 ", want %" PRIu32, got_height,
                want_height);
  }
  return NULL;
}

  // ---------------- Mimic Tests

#ifdef WUFFS_MIMIC

  // No mimic tests.

#endif  // WUFFS_MIMIC

  // ---------------- WBMP Benches

  // No WBMP benches.

  // ---------------- Mimic Benches

#ifdef WUFFS_MIMIC

  // No mimic benches.

#endif  // WUFFS_MIMIC

// ---------------- Manifest

// The empty comments forces clang-format to place one element per line.
proc tests[] = {

    test_wuffs_wbmp_decode_frame_config,  //
    test_wuffs_wbmp_decode_image_config,  //
    test_wuffs_wbmp_decode_interface,     //

#ifdef WUFFS_MIMIC

// No mimic tests.

#endif  // WUFFS_MIMIC

    NULL,
};

// The empty comments forces clang-format to place one element per line.
proc benches[] = {

// No WBMP benches.

#ifdef WUFFS_MIMIC

// No mimic benches.

#endif  // WUFFS_MIMIC

    NULL,
};

int main(int argc, char** argv) {
  proc_package_name = "std/wbmp";
  return test_main(argc, argv, tests, benches);
}
